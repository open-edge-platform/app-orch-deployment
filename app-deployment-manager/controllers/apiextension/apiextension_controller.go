// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package apiextension

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet"
	"os"
	"path/filepath"
	"strconv"
	"time"

	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/config"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/ingresshandler"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/istio"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/patch"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/randomtoken"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/gitclient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/vault"
)

var (
	Clock clock.Clock = clock.RealClock{}

	ownedResourcePredicate = predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool {
			// no action
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// process if API extension annotation exists
			_, ok := e.Object.GetAnnotations()[v1beta1.APIExtensionAnnotation]
			return ok
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// process if the original object has API extension annotation
			_, ok := e.ObjectOld.GetAnnotations()[v1beta1.APIExtensionAnnotation]
			return ok
		},
		GenericFunc: func(_ event.GenericEvent) bool {
			// no action
			return false
		},
	}
)

// Reconciler reconciles a APIExtension object
type Reconciler struct {
	Client client.Client
	Scheme *runtime.Scheme

	recorder   record.EventRecorder
	cfg        *config.APIExtensionConfig
	ingHandler ingresshandler.IngressHandler
}

type APIAgentRegistrationSecret struct {
	Token string `yaml:"token"`
}

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=get;list;watch
// +kubebuilder:rbac:groups=traefik.containo.us,resources=ingressroutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=traefik.containo.us,resources=ingressroutes/status,verbs=get
// +kubebuilder:rbac:groups=traefik.containo.us,resources=middlewares,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=traefik.containo.us,resources=middlewares/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get
// +kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=apiextensions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=apiextensions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=apiextensions/finalizers,verbs=update
// +kubebuilder:rbac:groups=fleet.cattle.io,resources=gitrepos,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, err := ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.APIExtension{}).
		Owns(&networkv1.Ingress{}, builder.WithPredicates(ownedResourcePredicate)).
		Owns(&corev1.Secret{}, builder.WithPredicates(ownedResourcePredicate)).
		Owns(&fleetv1alpha1.GitRepo{}, builder.WithPredicates(ownedResourcePredicate)).
		Build(r)
	if err != nil {
		return fmt.Errorf("failed to setup apiextension controller (%v)", err)
	}

	r.recorder = mgr.GetEventRecorderFor("controller")
	r.cfg = config.GetAPIExtensionConfig()
	r.ingHandler, err = ingresshandler.New(r.Scheme, r.Client)
	if err != nil {
		return fmt.Errorf("failed to setup apiextension controller (%v)", err)
	}
	r.ingHandler.Init()

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := log.FromContext(ctx)

	a := &v1beta1.APIExtension{}
	if err := r.Client.Get(ctx, req.NamespacedName, a); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return early
			return ctrl.Result{}, nil
		}
		// Error reading object, requeue the request
		return ctrl.Result{}, err
	}

	patchHelper, err := patch.NewPatchHelper(a, r.Client)
	if err != nil {
		log.Error(err, "error creating patch helper")
		return ctrl.Result{}, err
	}

	defer func() {
		// Always reconcile Status.State field after each reconciliation
		r.reconcileState(ctx, a)

		// Patch ObservedGeneration if the reconciliation completed successfully
		// We still run reconcile loop when CR isn't updated to to handle the case where
		// resources owned by the apiextension are deleted or modified
		patchOpts := []patch.Option{}
		if reterr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		}

		// Always attempt to patch the apiextension object after each reconciliation
		if err := patchHelper.Patch(ctx, a, patchOpts...); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	// Add finalizer first to avoid the race condition between init and delete
	if !cutil.ContainsFinalizer(a, v1beta1.APIExtensionFinalizer) {
		cutil.AddFinalizer(a, v1beta1.APIExtensionFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop if the object is marked as deleted
	if !a.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.delete(ctx, a)
	}

	// Handle normal reconciliation loop
	result, err := r.reconcile(ctx, a)
	if err != nil {
		r.recorder.Eventf(a, corev1.EventTypeWarning, "ReconcileError", "Reconcile failed: %v", err)
	}

	return result, err
}

func (r *Reconciler) delete(ctx context.Context, a *v1beta1.APIExtension) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if cutil.ContainsFinalizer(a, v1beta1.APIExtensionFinalizer) {
		log.Info("performing finalizer operations for the apiextension")

		repo, err := gitclient.NewGitClient(string(a.UID))
		if err != nil {
			// Return with error for retry
			return ctrl.Result{}, err
		}

		exists, err := repo.ExistsOnRemote()
		if err != nil {
			return ctrl.Result{}, err
		}
		// Remove the repo is exists
		if exists {
			if err := repo.Delete(); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	cutil.RemoveFinalizer(a, v1beta1.APIExtensionFinalizer)
	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, a *v1beta1.APIExtension) (ctrl.Result, error) {
	phases := []func(context.Context, *v1beta1.APIExtension) (ctrl.Result, error){
		r.reconcileToken,
		r.reconcileAgentRepo,
		r.reconcileAgent,
		r.reconcileIngress,
	}

	errs := []error{}
	for _, phase := range phases {
		// Call the inner reconciliation methods
		_, err := phase(ctx, a)
		if err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			continue
		}
	}

	return ctrl.Result{}, kerrors.NewAggregate(errs)
}

func (r *Reconciler) reconcileState(_ context.Context, a *v1beta1.APIExtension) {
	preState := a.Status.State

	// Set not ready state if the state is empty
	if a.Status.State == "" {
		a.Status.State = v1beta1.Down
	}

	if HasCondition(a, metav1.Condition{
		Type:   string(v1beta1.TokenReady),
		Status: metav1.ConditionTrue,
	}) && HasCondition(a, metav1.Condition{
		Type:   string(v1beta1.AgentRepoReady),
		Status: metav1.ConditionTrue,
	}) && HasCondition(a, metav1.Condition{
		Type:   string(v1beta1.AgentReady),
		Status: metav1.ConditionTrue,
	}) && HasCondition(a, metav1.Condition{
		Type:   string(v1beta1.IngressReady),
		Status: metav1.ConditionTrue,
	}) {
		// Set the state Running only if all conditions are ready
		a.Status.State = v1beta1.Running
	} else {
		a.Status.State = v1beta1.Down
	}

	if !a.DeletionTimestamp.IsZero() {
		a.Status.State = v1beta1.Terminating
	}

	// Record the event if the status has changed
	if preState != a.Status.State {
		if a.Status.State == v1beta1.Down {
			r.recorder.Eventf(a, corev1.EventTypeWarning,
				string(a.Status.State), "API extension is %q",
				a.Status.State)
		} else {
			r.recorder.Eventf(a, corev1.EventTypeNormal,
				string(a.Status.State), "API extension is %q",
				a.Status.State)
		}
	}
}

func (r *Reconciler) reconcileToken(ctx context.Context, a *v1beta1.APIExtension) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure auth token secret exists
	tokenSecretName := fmt.Sprintf("%s-token", a.Name)
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: a.Namespace,
		Name:      tokenSecretName,
	}, secret)

	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to get secret resource")
		// Error when listing secret resources
		// Return with error for another try
		return ctrl.Result{}, err
	}

	// Secret exists, set condition and return
	// TODO: handle the case where secret data is modified
	if err == nil {
		SetCondition(a, v1beta1.TokenReady, metav1.ConditionTrue)
		return ctrl.Result{}, nil
	}

	// Token secret does not exist
	SetCondition(a, v1beta1.TokenReady, metav1.ConditionFalse)

	// Generate a new auth token if it hasn't yet
	tokenRef := &a.Status.TokenSecretRef
	tokenExpired := r.tokenExpired(ctx, tokenRef.Timestamp)
	if !tokenRef.IsValid() || tokenExpired {
		token, err := randomtoken.Generate()
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to generate token for %q (%v)", a.Name, err)
		}

		tokenRef.Name = tokenSecretName
		tokenRef.GeneratedToken = token
		tokenRef.Timestamp = time.Now().String()

		r.recorder.Eventf(a, corev1.EventTypeNormal, "Reconciling",
			"Generated new token with client id %q, timestamp %q", a.Name, tokenRef.Timestamp)
	}

	secret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      tokenSecretName,
		Namespace: a.Namespace,
		Annotations: map[string]string{
			v1beta1.APIExtensionAnnotation: a.Name,
		},
	},
		Data: map[string][]byte{
			"token": []byte(tokenRef.GeneratedToken),
		},
	}

	if tokenExpired {
		_ = r.Client.Delete(ctx, secret)
	}

	if err := ctrl.SetControllerReference(a, secret, r.Scheme); err != nil {
		// Secret object is owned by the associated apiextension
		// and cannot be shared by other resources
		// This secret is removed by GC when owner apiextension is deleted
		log.Error(err, "failed to set cluster group owner")
		return ctrl.Result{}, err
	}

	if err := r.Client.Create(ctx, secret); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create secret %q (%v)", secret.Name, err)
	}

	// Recored the event of successful token creation for the APIextension
	r.recorder.Eventf(a, corev1.EventTypeNormal,
		"Reconciling", "Created Secret %q",
		secret.Name)
	SetCondition(a, v1beta1.TokenReady, metav1.ConditionTrue)

	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileAgentRepo(_ context.Context, a *v1beta1.APIExtension) (ctrl.Result, error) {
	agentRepo, err := gitclient.NewGitClient(string(a.UID))
	if err != nil {
		return ctrl.Result{}, err
	}

	exists, err := agentRepo.ExistsOnRemote()
	if err != nil {
		return ctrl.Result{}, err
	}

	// Repository already exists, return early
	// TODO: should check the contents?
	if exists {
		SetCondition(a, v1beta1.AgentRepoReady, metav1.ConditionTrue)
		return ctrl.Result{}, nil
	}

	SetCondition(a, v1beta1.AgentRepoReady, metav1.ConditionFalse)

	// Git repository for API Agent does not exist
	// Create a new repo
	basedir := filepath.Join("/tmp", string(a.UID))
	os.RemoveAll(basedir)

	if err := agentRepo.Initialize(basedir); err != nil {
		return ctrl.Result{}, err
	}

	// Generate agent fleet configs, commit and push
	if err := r.generateAgentConfig(a, basedir); err != nil {
		return ctrl.Result{}, err
	}
	if err := agentRepo.CommitFiles(); err != nil {
		return ctrl.Result{}, err
	}
	if err := agentRepo.PushToRemote(); err != nil {
		return ctrl.Result{}, err
	}

	// Clean up temporary dir
	os.RemoveAll(basedir)

	r.recorder.Eventf(a, corev1.EventTypeNormal, "Reconciling",
		"Initialized git repository %q for api-agent", a.UID)

	SetCondition(a, v1beta1.AgentRepoReady, metav1.ConditionTrue)
	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileAgent(ctx context.Context, a *v1beta1.APIExtension) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	repoURL, err := gitclient.GetRemoteURLWithCreds(string(a.UID))
	if err != nil {
		// Failed to get Fleet git repo url for API agent
		// Return with error to retry
		return ctrl.Result{}, err
	}

	gitrepo := &fleetv1alpha1.GitRepo{}
	err = r.Client.Get(ctx, client.ObjectKey{
		Namespace: a.Namespace,
		Name:      fmt.Sprintf("%s-api-agent", a.Name),
	}, gitrepo)

	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	// GitRepo exists, update the condition and status
	// TODO: check if spec was modified
	if err == nil {
		if ready, msg := isGitRepoReady(gitrepo); ready {
			SetCondition(a, v1beta1.AgentReady, metav1.ConditionTrue)
		} else {
			SetCondition(a, v1beta1.AgentReady, metav1.ConditionFalse)
			if msg != "" {
				r.recorder.Eventf(a, corev1.EventTypeWarning, "Reconciling", msg)
			}
		}
		return ctrl.Result{}, nil
	}

	helmSecretName := os.Getenv("API_AGENT_HELM_SECRET_NAME")
	if helmSecretName != "" {
		err := r.createHelmSecret(ctx, a, helmSecretName)
		if err != nil {
			log.Error(err, "failed to create/update secret")
			return ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("helmSecretName: %s", helmSecretName))
	}

	// GitRepo does not exist, create a new one
	SetCondition(a, v1beta1.AgentReady, metav1.ConditionFalse)

	gitrepo = &fleetv1alpha1.GitRepo{ObjectMeta: metav1.ObjectMeta{
		Namespace: a.Namespace,
		Name:      fmt.Sprintf("%s-api-agent", a.Name),
		Annotations: map[string]string{
			v1beta1.APIExtensionAnnotation: a.Name,
		},
		Labels: map[string]string{
			string(v1beta1.BundleName): fmt.Sprintf("%s-api-agent", a.Name),
			string(v1beta1.BundleType): fleet.BundleTypeApp.String(),
		},
	},
		Spec: fleetv1alpha1.GitRepoSpec{
			Repo:           repoURL,
			Paths:          []string{"api-agent"},
			HelmSecretName: helmSecretName,
			Targets: []fleetv1alpha1.GitTarget{
				{
					Name: "match", ClusterSelector: &metav1.LabelSelector{
						MatchLabels: a.Spec.AgentClusterLabels,
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(a, gitrepo, r.Scheme); err != nil {
		// GitRepo object is owned by the associated apiextension
		// and cannot be shared by other resources
		// This GitRepo is removed by GC when owner apiextension is deleted
		log.Error(err, "failed to set cluster group owner")
		return ctrl.Result{}, err
	}

	if err := r.Client.Create(ctx, gitrepo); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create gitrepo %q (%v)", gitrepo.Name, err)
	}

	// Record the event of successful ingress creation for the APIextension
	r.recorder.Eventf(a, corev1.EventTypeNormal,
		"Reconciling", "Created Fleet GitRepo %q",
		gitrepo.Name)

	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileIngress(ctx context.Context, a *v1beta1.APIExtension) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure route is created for each proxy endpoint
	for _, e := range a.Spec.ProxyEndpoints {
		_, err := r.ingHandler.GetRoute(ctx, client.ObjectKey{
			Namespace: a.Namespace,
			Name:      e.ServiceName,
		})

		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "failed to get ingress resource")
			return ctrl.Result{}, err
		}

		// Ingress exists, continue
		// TODO: handle the case where ingress is update
		// TODO: check the ingress status and update condition
		if err == nil {
			continue
		}

		SetCondition(a, v1beta1.IngressReady, metav1.ConditionFalse)
		if err := r.ingHandler.CreateRoute(ctx, a, e); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create ingress %q (%v)", e.ServiceName, err)
		}

		externalPath := r.getExternalEndpoint(a.Spec.APIGroup, e.Path)
		a.Status.AppliedAPIEndpoints = append(a.Status.AppliedAPIEndpoints, externalPath)

		// Record the event of successful ingress creation for the APIextension
		r.recorder.Eventf(a, corev1.EventTypeNormal,
			"Reconciling", "Created Ingress %q with external path %q",
			e.ServiceName, externalPath)
	}

	SetCondition(a, v1beta1.IngressReady, metav1.ConditionTrue)
	return ctrl.Result{}, nil
}

func (r *Reconciler) getExternalEndpoint(apiGroup v1beta1.APIGroup, path string) string {
	return fmt.Sprintf("/%s.%s/%s/%s",
		apiGroup.Name,
		r.cfg.APIGroupDomain,
		apiGroup.Version,
		path)
}

func (r *Reconciler) generateAgentConfig(a *v1beta1.APIExtension, basedir string) error {
	agentConfig := struct {
		Conf   v1beta1.APIAgentConfig
		Secret APIAgentRegistrationSecret
	}{
		Conf: v1beta1.APIAgentConfig{
			AgentId:  a.Name,
			ProxyURL: r.cfg.APIProxyURL},
		Secret: APIAgentRegistrationSecret{
			Token: a.Status.TokenSecretRef.GeneratedToken,
		},
	}
	values, err := yaml.Marshal(&agentConfig)
	if err != nil {
		return err
	}

	initFleetdir := filepath.Join(basedir, "api-agent-init")
	initFleetconfig := fleet.Config{
		Name:             fmt.Sprintf("%s-api-agent-init", a.Name),
		DefaultNamespace: r.cfg.APIAgentNamespace,
		Labels: fleet.DeployLabels{
			AppName:    fmt.Sprintf("%s-api-agent", a.Name),
			BundleType: fleet.BundleTypeInit.String(),
		},
		NamespaceLabels: map[string]string{istio.IstioInjectionLabelKey: istio.IstioInjectionLabelValueEnabled},
	}
	if err := fleet.WriteFleetConfig(initFleetdir, initFleetconfig); err != nil {
		return err
	}

	fleetdir := filepath.Join(basedir, "api-agent")
	if err := utils.WriteFile(fleetdir, "overrides.yaml", values); err != nil {
		return err
	}

	fleetconfig := fleet.Config{
		Name:             fmt.Sprintf("%s-api-agent", a.Name),
		DefaultNamespace: r.cfg.APIAgentNamespace,
		Labels: fleet.DeployLabels{
			AppName:    fmt.Sprintf("%s-api-agent", a.Name),
			BundleType: fleet.BundleTypeApp.String(),
		},
		Helm: fleet.HelmApp{
			ReleaseName: "api-agent",
			Repo:        r.cfg.APIAgentChartRepo,
			Chart:       r.cfg.APIAgentChart,
			Version:     r.cfg.APIAgentChartVersion,
			ValuesFiles: []string{"overrides.yaml"},
		},
		NamespaceLabels: map[string]string{istio.IstioInjectionLabelKey: istio.IstioInjectionLabelValueEnabled},
		DependsOn:       []fleet.DependsOnItem{},
	}
	if err := fleet.WriteFleetConfig(fleetdir, fleetconfig); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createHelmSecret(ctx context.Context, a *v1beta1.APIExtension, helmSecretName string) error {
	data := make(map[string][]byte)
	log := log.FromContext(ctx)

	// Get cacerts from Vault
	vaultManager := vault.NewManager(utils.GetSecretServiceEndpoint(), utils.GetServiceAccount(), utils.GetSecretServiceMount())
	vaultClient, err := vaultManager.GetVaultClient(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		if err := vaultManager.Logout(context.Background(), vaultClient); err != nil {
			log.Info(fmt.Sprintf("Error logging out from Vault: %v\n", err))
		}
	}()

	if user, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceHarborServicePath(), utils.GetSecretServiceHarborServiceKVKeyUsername()); err == nil {
		data["username"] = []byte(user)
	}

	if passwd, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceGitServicePath(), utils.GetSecretServiceGitServiceKVKeyPassword()); err == nil {
		data["password"] = []byte(passwd)
	}

	if caCertificateBase64, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceHarborServicePath(), utils.GetSecretServiceHarborServiceKVKeyCert()); err == nil {
		// Decode the CA certificate from base64
		caCertificateData, err := base64.StdEncoding.DecodeString(caCertificateBase64)
		if err != nil {
			log.Info(fmt.Sprintf("Error decoding CA certificate: %v\n", err))
			return err
		}
		log.Info(fmt.Sprintf("createHelmSecret, vault cacerts: %s", caCertificateData))
		data["cacerts"] = caCertificateData
	}

	// Create or update helm secret
	namespace := a.Namespace
	secret := v1.Secret{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      helmSecretName,
	}
	err = r.Client.Get(ctx, key, &secret)
	if client.IgnoreNotFound(err) != nil {
		log.Info(fmt.Sprintf("Error getting secret: %v\n", err))
		return err
	}
	if apierrors.IsNotFound(err) {
		// Need to create a new Secret
		secret = v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helmSecretName,
				Namespace: namespace,
				// Need to set owner reference so this Secret gets deleted
				OwnerReferences: []metav1.OwnerReference{},
			},
			Type: v1.SecretTypeOpaque,
			Data: data,
		}

		// Secret is owned by the APIExtension
		if err := ctrl.SetControllerReference(a, &secret, r.Scheme); err != nil {
			log.Error(err, "Failed to set Secret owner")
			return err
		}

		if err := r.Client.Create(ctx, &secret); err != nil {
			log.Error(err, "Failed to create Helm secret")
			return err
		}
		log.Info(fmt.Sprintf("Created Helm secret %s", helmSecretName))
	} else {
		// Update existing Secret

		secret.Data = data

		if err := r.Client.Update(ctx, &secret); err != nil {
			log.Error(err, "Failed to update Helm secret")
			return err
		}
		log.Info(fmt.Sprintf("Updated Helm secret %s", helmSecretName))
	}

	return nil
}

func (r *Reconciler) tokenExpired(ctx context.Context, generatedTimestamp string) bool {
	log := log.FromContext(ctx)

	// Calculate (in secs) the expected expiry timestamp
	expiryDays, err := strconv.Atoi(r.cfg.TokenExpiryDays)
	if err != nil {
		log.Error(err, "Invalid value for token expiry days:")
		return false
	}
	expirySecs := int64(expiryDays) * 86400

	generatedTime, err := time.Parse(time.RFC3339, generatedTimestamp)
	if err != nil {
		log.Error(err, "Error parsing time")
		return false
	}

	expiryTime := generatedTime.Add(time.Duration(expirySecs) * time.Second)
	now := time.Now()
	expired := expiryTime.Before(time.Now())
	if expired {
		log.Info(fmt.Sprintf("token expired, current time:%s, expiry time:%s", now, expiryTime))
	}
	return expired
}

func HasCondition(a *v1beta1.APIExtension, c metav1.Condition) bool {
	if a == nil {
		return false
	}
	existingConditions := a.Status.Conditions
	for _, cond := range existingConditions {
		if c.Type == cond.Type && c.Status == cond.Status {
			return true
		}
	}

	return false
}

func SetCondition(a *v1beta1.APIExtension, condType v1beta1.APIExtensionConditionType, condStatus metav1.ConditionStatus) {
	nowTime := metav1.NewTime(Clock.Now())
	condition := metav1.Condition{
		Type:               string(condType),
		Status:             condStatus,
		LastTransitionTime: nowTime,
		Reason:             fmt.Sprintf("%s:%s", condType, condStatus),
		Message:            fmt.Sprintf("%s:%s", condType, condStatus),
	}

	for i, c := range a.Status.Conditions {
		// Skip unrelated conditions
		if c.Type != string(condType) {
			continue
		}
		// If this update doesn't contain a state transition, don't update
		// the conditions LastTransitionTime to Now()
		if c.Status == condStatus {
			condition.LastTransitionTime = c.LastTransitionTime
		}
		// Overwrite the existing condition
		a.Status.Conditions[i] = condition
		return
	}

	// If not found an existing condition of this type, simply insert
	// the new condition into the slice
	a.Status.Conditions = append(a.Status.Conditions, condition)
}

func isGitRepoReady(g *fleetv1alpha1.GitRepo) (bool, string) {
	if g == nil {
		return false, "GitRepo does not exists"
	}

	conditions := g.Status.Conditions
	for _, cond := range conditions {
		if cond.Type == "Ready" {
			if cond.Status == corev1.ConditionTrue {
				if g.Status.ReadyClusters == 0 {
					return false, "Deployment does not have minimum available target clusters"
				}
				return true, ""
			}
			return false, cond.Message
		}
	}
	return false, ""
}
