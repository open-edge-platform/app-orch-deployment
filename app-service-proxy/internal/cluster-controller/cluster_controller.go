// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	clusterv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/gitclient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/fleet"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/vault"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/auth"
)

const (
	FinalizerServiceProxy = "app.orchestrator.io/service-proxy"
	AgentIDLabel          = "global.fleet.clusterLabels.cluster.orchestration.io/cluster-id"
)

var (
	// Predicates that passes only when the non-local cluster has created or deleted
	clusterCreateDeletePredicate = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetName() != "local"
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Object.GetName() != "local"
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return (e.ObjectNew.GetName() != "local") && !e.ObjectNew.GetDeletionTimestamp().IsZero()
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return (e.Object.GetName() != "local") && !e.Object.GetDeletionTimestamp().IsZero()
		},
	}
)

type AgentValues struct {
	Image AgentValuesImage `yaml:"image"`
	Conf  AgentValuesConf  `yaml:"conf"`
}

type AgentValuesConf struct {
	ProxyDomain          string `yaml:"proxyDomain"`
	ProxyServerURL       string `yaml:"proxyServerURL"`
	ProxyServerCA        string `yaml:"proxyServerCA,omitempty"`
	AgentID              string `yaml:"agentId"`
	AgentToken           string `yaml:"agentToken,omitempty"`
	InsecureSkipVerify   string `yaml:"insecureSkipVerify,omitempty"`
	AgentTokenSecretName string `yaml:"agentTokenSecretName,omitempty"`
}

type AgentValuesImage struct {
	Registry AgentValuesImageRegistry `yaml:"registry"`
}

type AgentValuesImageRegistry struct {
	Name string `yaml:"name"`
}

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	proxyDomain          string
	proxyServerURL       string
	agentRegistry        string
	agentChart           string
	agentChartVersion    string
	agentChartRepoSecret string
	agentTargetNamespace string
	gitRepoName          string // The name used for both git repository and Fleet GitRepo object
	gitRemoteRepoName    string
	gitRepoURL           string
	gitClient            gitclient.ClientCreator
	gitClientSecret      string

	agentValues  []byte
	vaultManager vault.Manager
}

//+kubebuilder:rbac:groups=app.orchestrator.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.orchestrator.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.orchestrator.io,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=fleet.cattle.io,resources=gitrepos,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=fleet.cattle.io,resources=bundles,verbs=get;list;watch
//+kubebuilder:rbac:groups=fleet.cattle.io,resources=bundledeployments,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) (err error) {
	if err = r.initReconciler(); err != nil {
		return fmt.Errorf("failed to setup cluster controller (%v)", err)
	}

	_, err = ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1beta1.Cluster{}, builder.WithPredicates(clusterCreateDeletePredicate)).
		Owns(&fleetv1alpha1.GitRepo{}).
		Build(r)

	if err != nil {
		return fmt.Errorf("failed to setup cluster controller (%v)", err)
	}

	return nil
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling", "cluster", req)

	c := &clusterv1beta1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, c); err != nil {
		// Error reading the object, requeue the request, unless error is "not found"
		if client.IgnoreNotFound(err) == nil {
			log.Info("Cluster not found")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add finalizer first to avoid the race condition between init and delete.
	if !cutil.ContainsFinalizer(c, FinalizerServiceProxy) {
		log.Info("New cluster, adding finalizer")
		cutil.AddFinalizer(c, FinalizerServiceProxy)
		if err := r.Update(ctx, c); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer (%s)", err)
		}
	}

	// Handle finalizers if the deletion timestamp is non-zero
	if !c.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("Cluster is deleted")
		return r.delete(ctx, c)
	}

	// Set agentValues if not set yet
	// TODO: Remove this one time process from reconcile loop using pre-created secret for CA
	// in the same namespace as controller and mount it as file
	if len(r.agentValues) == 0 {
		if err := r.setAgentValues(ctx); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set agent values (%s)", err)
		}
	}

	// Reconcile fleet configuration in git repository
	if err := r.reconcileFleetConfig(ctx, c); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile Fleet configs (%s)", err)
	}

	// Reconcile Fleet GitRepo object
	if err := r.reconcileFleetGitRepo(ctx, c); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile GitRepo (%s)", err)
	}

	log.Info("Reconciliation successful")
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) initReconciler() error {
	// Initialize agent chart configurations from the environment variables
	if r.agentRegistry = os.Getenv("AGENT_REGISTRY"); r.agentRegistry == "" {
		return fmt.Errorf("AGENT_REGISTRY is not set")
	}

	if r.agentChart = os.Getenv("AGENT_CHART"); r.agentChart == "" {
		return fmt.Errorf("AGENT_CHART is not set")
	}

	if r.agentChartVersion = os.Getenv("AGENT_CHART_VERSION"); r.agentChartVersion == "" {
		return fmt.Errorf("AGENT_CHART_VERSION is not set")
	}

	if r.agentTargetNamespace = os.Getenv("AGENT_TARGET_NAMESPACE"); r.agentTargetNamespace == "" {
		return fmt.Errorf("AGENT_TARGET_NAMESPACE is not set")
	}

	if r.proxyDomain = os.Getenv("PROXY_DOMAIN"); r.proxyDomain == "" {
		return fmt.Errorf("PROXY_DOMAIN is not set")
	}

	if r.proxyServerURL = os.Getenv("PROXY_SERVER_URL"); r.proxyServerURL == "" {
		return fmt.Errorf("PROXY_SERVER_URL is not set")
	}

	// Chart repo secret will not be used in most of the case where Release
	// Service Proxy is available.
	// TODO: Consider to remove this configuration and repo secret creation step.
	r.agentChartRepoSecret = os.Getenv("AGENT_CHART_REPO_SECRET")

	// Initialize remote git repository configurations from the environment variables
	if r.gitRepoName = os.Getenv("GIT_REPO_NAME"); r.gitRepoName == "" {
		r.gitRepoName = "orchestrator-service-proxy"
	}

	// Create unique git repository name per environment to prevent conflict when multiple
	// environments share the same Git server
	var urlTemp *url.URL
	var err error
	if urlTemp, err = url.Parse(r.proxyServerURL); err != nil {
		return fmt.Errorf("failed to parse proxyServerURL (%v)", err)
	}

	r.gitRemoteRepoName = fmt.Sprintf("%s-%s", "app-service-proxy", urlTemp.Host)

	r.gitClientSecret = os.Getenv("GIT_CLIENT_SECRET")
	r.gitRepoURL = ""
	r.gitClient = nil

	r.vaultManager = vault.NewManager()
	return nil
}

func (r *ClusterReconciler) delete(ctx context.Context, c *clusterv1beta1.Cluster) (ctrl.Result, error) {
	var err error
	var gc gitclient.Repository

	log := log.FromContext(ctx)

	// remove vault token
	if err := r.vaultManager.DeleteToken(ctx, c.Name); err != nil {
		return ctrl.Result{}, err
	}

	// preliminaries - setup gitclient and gitrepo
	basedir := filepath.Join("/tmp", r.gitRepoName)
	os.RemoveAll(basedir)

	if r.gitRepoURL == "" {
		r.gitRepoURL, err = gitclient.GetRemoteURLWithCreds(r.gitRemoteRepoName)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to generate git repository URL (%v)", err)
		}
	}

	if r.gitClient == nil {
		r.gitClient = gitclient.NewGitClient
	}

	if gc, err = r.gitClient(r.gitRemoteRepoName); err != nil {
		return ctrl.Result{}, err
	}

	// clone gitrepo
	if err := gc.Clone(basedir); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to uninstall app-service-proxy-agent (%s);"+
			"failed to clone git repositor: err - %+v", c.Name, err)
	}

	// Update fleet.yaml
	fleetConfig, err := fleet.ReadFleetYAMLFile(basedir)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to uninstall app-service-proxy-agent (%s);"+
			"failed to read fleet.yaml file: err - %+v", c.Name, err)
	}

	// remove targetCustomization entry
	tcName := fleet.GetTargetCustomizationName(c.Name)
	for idx := range fleetConfig.TargetCustomizations {
		if fleetConfig.TargetCustomizations[idx].Name == tcName {
			fleetConfig.TargetCustomizations = append(fleetConfig.TargetCustomizations[:idx], fleetConfig.TargetCustomizations[idx+1:]...)
			break
		}
	}

	err = fleet.WriteFleetYAML(basedir, fleetConfig)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to uninstall app-service-proxy-agent (%s);"+
			"failed to update fleet.yaml file: err - %+v", c.Name, err)
	}

	// delete kustomization directory
	err = os.RemoveAll(path.Join(basedir, fleet.FleetOverlaysDirName, tcName))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to uninstall app-service-proxy-agent (%s);"+
			"failed to remove Kustomization directory: err - %+v", c.Name, err)
	}

	//commit/push to git repository
	if err := gc.CommitFiles(); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to uninstall app-service-proxy-agent (%s);"+
			"failed to commit changes: err - %+v", c.Name, err)
	}
	if err := gc.PushToRemote(); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to uninstall app-service-proxy-agent (%s);"+
			"failed to push commit: err - %+v", c.Name, err)
	}

	cutil.RemoveFinalizer(c, FinalizerServiceProxy)

	log.Info("Removed finalizer, updating cluster")
	return ctrl.Result{}, r.Client.Update(ctx, c)
}

func (r *ClusterReconciler) reconcileFleetConfig(ctx context.Context, c *clusterv1beta1.Cluster) error {
	var gc gitclient.Repository
	var repoExists bool
	var err error

	// Clean up local repo path if exists
	basedir := filepath.Join("/tmp", r.gitRepoName)
	os.RemoveAll(basedir)

	if r.gitRepoURL == "" {
		r.gitRepoURL, err = gitclient.GetRemoteURLWithCreds(r.gitRemoteRepoName)
		if err != nil {
			return fmt.Errorf("failed to generate git repository URL (%v)", err)
		}
	}

	if r.gitClient == nil {
		r.gitClient = gitclient.NewGitClient
	}

	if gc, err = r.gitClient(r.gitRemoteRepoName); err != nil {
		return err
	}

	repoExists, err = gc.ExistsOnRemote()
	if err != nil {
		return err
	}

	// Clone remote repo to basedir if the repo exists, otherwise initialize the repo from basedir
	if repoExists {
		// if repo exists, clone first
		if err := gc.Clone(basedir); err != nil {
			return err
		}
	} else {
		// if repo does not exist, init git repository
		if err := gc.Initialize(basedir); err != nil {
			return err
		}
	}

	// create overlays directory
	err = os.Mkdir(filepath.Join(basedir, fleet.FleetOverlaysDirName), 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	tokenStr, err := auth.GenerateToken()
	if err != nil {
		return err
	}

	ttl, err := auth.GetTokenTTLHours()
	if err != nil {
		return err
	}

	// put token to vault
	if err := r.vaultManager.PutToken(ctx, c.Name, tokenStr, ttl); err != nil {
		return fmt.Errorf("error putting secret in Vault for %s: %s", c.Name, err)
	}

	token, err := r.vaultManager.GetToken(ctx, c.Name)
	if err != nil {
		return fmt.Errorf("error getting secret from Vault right after putting it for %s: %s", c.Name, err)
	}
	// Generate fleet configurations for the service-proxy agent,
	// commit and push to the remote git repository
	if err := utils.WriteFile(basedir, fleet.FleetValuesFileName, r.agentValues); err != nil {
		return err
	}

	// get fleet.yaml
	fleetConfig, err := fleet.GetFleetYAML(basedir, r.agentTargetNamespace, r.gitRepoName, r.agentChart, r.agentChartVersion)
	if err != nil {
		return err
	}

	// set targetCustomization name formatted with prefix app-service-proxy-agent and cluster name
	tcName := fleet.GetTargetCustomizationName(c.Name)

	// Add targetCustomization and Kustomize config for the agentToken

	tc := fleetv1alpha1.BundleTarget{
		Name:        tcName,
		ClusterName: c.Name,
	}

	// check if targetCustomizations is already made or not
	isAdded := false
	for _, existingTC := range fleetConfig.TargetCustomizations {
		if existingTC.Name == tc.Name && existingTC.ClusterName == tc.ClusterName {
			isAdded = true
			break
		}
	}
	if !isAdded {
		tcHelm := &fleetv1alpha1.HelmOptions{}
		tc.Helm = tcHelm
		tc.Kustomize = &fleetv1alpha1.KustomizeOptions{
			Dir: filepath.Join(fleet.FleetOverlaysDirName, tcName),
		}

		fleetConfig.TargetCustomizations = append(fleetConfig.TargetCustomizations, tc)
		err = os.Mkdir(filepath.Join(basedir, fleet.FleetOverlaysDirName, tcName), 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}

	// get kustomization.yaml file
	kustomizationConfig := fleet.GetKustomizationYAML()
	kustomizationConfig.Resources = append(kustomizationConfig.Resources, fleet.TokenSecretFileName)

	// write fleet.yaml file
	err = fleet.WriteFleetYAML(basedir, fleetConfig)
	if err != nil {
		return err
	}

	// write kustomization.yaml
	err = fleet.WriteKustomization(filepath.Join(basedir, fleet.FleetOverlaysDirName, tcName), kustomizationConfig)
	if err != nil {
		return err
	}

	// write token secret
	err = fleet.WriteSecretToken(basedir, tcName, r.agentTargetNamespace, token.Value,
		fmt.Sprintf("%d", token.TTLHours), token.UpdatedTime.Format(time.DateTime))
	if err != nil {
		return err
	}

	if err := gc.CommitFiles(); err != nil {
		return err
	}

	if err := gc.PushToRemote(); err != nil {
		return err
	}

	return nil
}

func (r *ClusterReconciler) reconcileFleetGitRepo(ctx context.Context, c *clusterv1beta1.Cluster) error {
	log := log.FromContext(ctx)

	gitRepo := &fleetv1alpha1.GitRepo{}
	err := r.Get(ctx, client.ObjectKey{Namespace: c.Namespace, Name: r.gitRepoName}, gitRepo)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("unexpected error to get GitRepo %s(%v)", r.gitRepoName, err)
	}

	if err == nil {
		// GitRepo already exists, check if URL changed
		// TODO: improve to ensure GitRepo spec has expected configs
		gitRepoURL, err := gitclient.GetRemoteURLWithCreds(r.gitRemoteRepoName)
		if err != nil {
			return err
		}

		r.gitRepoURL = gitRepoURL

		// If URL changed, update the gitRepo with new URL
		if gitRepo.Spec.Repo != r.gitRepoURL {
			log.Info("Detected Git repo URL change for Deployment")
			gitRepo.Spec.Repo = r.gitRepoURL

			caCert := utils.GetGitCaCert()
			// Set the CABundle for the GitRepo if one was provided
			if caCert != "" {
				gitRepo.Spec.CABundle = []byte(caCert)
			}

			gitRepo.Spec.ClientSecretName = r.gitClientSecret
			if err := r.Client.Update(ctx, gitRepo); err != nil {
				return fmt.Errorf("GitRepo URL update failed(%v)", err)
			}

			log.Info("Completed updating GitRepo")
		}
	} else {
		caCert := utils.GetGitCaCert()
		err := r.createSecrets(ctx, c)
		if err != nil {
			return fmt.Errorf("Create secrets for gitea failed")
		}
		// GitRepo does not exist, create a new one
		gitRepo = &fleetv1alpha1.GitRepo{
			ObjectMeta: metav1.ObjectMeta{Namespace: c.Namespace, Name: r.gitRepoName},
			Spec: fleetv1alpha1.GitRepoSpec{
				HelmSecretName:   r.agentChartRepoSecret,
				Repo:             r.gitRepoURL,
				ClientSecretName: r.gitClientSecret,
				// This redundant field is required to deploy to all clusters
				Targets: []fleetv1alpha1.GitTarget{
					{
						ClusterSelector: &metav1.LabelSelector{},
					},
				},
			},
		}
		// Set the CABundle for the GitRepo if one was provided
		if caCert != "" {
			gitRepo.Spec.CABundle = []byte(caCert)
		}
		if err = r.Client.Create(ctx, gitRepo); err != nil {
			return fmt.Errorf("failed to create GitRepo object(%v)", err)
		}
	}

	return nil
}

func (r *ClusterReconciler) createSecrets(ctx context.Context, c *clusterv1beta1.Cluster) (err error) {
	data, err := r.vaultManager.GetGitRepoCred(ctx)
	if err != nil {
		return fmt.Errorf("Get Git repo credentials failed(%v)", err)
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.gitClientSecret,
			Namespace: c.Namespace,
			// Need to set owner reference so this Secret gets deleted
			OwnerReferences: []metav1.OwnerReference{},
		},
		Type: corev1.SecretTypeBasicAuth,
	}

	secret.StringData = data

	if err = r.Client.Create(ctx, &secret); err != nil {
		return fmt.Errorf("failed to create secret object(%v)", err)
	}
	/*
		err := utils.UpdateNsLabels(ctx, s, nsName)
		if err != nil {
			utils.LogActivity(ctx, "create", "ADM", "cannot update labels on namespace "+nsName+" "+fmt.Sprintf("%v", err))
			return err
		}

		err = utils.CreateRoleBinding(ctx, s, nsName, rbName, remoteNsName)
		if err != nil {
			utils.LogActivity(ctx, "create", "ADM", "cannot create rolebinding "+rbName+" "+fmt.Sprintf("%v", err))
			return err
		}
	*/
	return nil
}
func (r *ClusterReconciler) setAgentValues(ctx context.Context) (err error) {
	// Create overriding values contents with mandatory fields
	values := AgentValues{
		Conf: AgentValuesConf{
			ProxyDomain:          r.proxyDomain,
			ProxyServerURL:       r.proxyServerURL,
			AgentID:              AgentIDLabel,
			AgentTokenSecretName: fleet.TokenSecretName,
		},
		Image: AgentValuesImage{
			Registry: AgentValuesImageRegistry{
				Name: r.agentRegistry,
			},
		},
	}

	// Set ProxyServerCA if provided
	if val := os.Getenv("PROXY_SERVER_CA"); val != "" {
		values.Conf.ProxyServerCA = val
	} else if os.Getenv("PROXY_SERVER_CA_SECRET_NAMESPACE") != "" && os.Getenv("PROXY_SERVER_CA_SECRET_NAME") != "" &&
		os.Getenv("PROXY_SERVER_CA_SECRET_KEY") != "" {
		// TODO: Pre-create a secret with CA in the same namespace (currently
		// exists in different namesapce) and mount it as file using volumeMount instead of
		// dynamically read it
		secret := &corev1.Secret{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Namespace: os.Getenv("PROXY_SERVER_CA_SECRET_NAMESPACE"),
			Name:      os.Getenv("PROXY_SERVER_CA_SECRET_NAME"),
		}, secret); err != nil {
			return err
		}
		values.Conf.ProxyServerCA = string(secret.Data[os.Getenv("PROXY_SERVER_CA_SECRET_KEY")])
	}

	// Set InsecureSkipVerify if provided
	if val := os.Getenv("INSECURE_SKIP_VERIFY"); val != "" {
		values.Conf.InsecureSkipVerify = val
	}

	if r.agentValues, err = yaml.Marshal(&values); err != nil {
		return fmt.Errorf("failed to generate default agent configs (%v)", err)
	}

	return nil
}
