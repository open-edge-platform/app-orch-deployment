// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet"
	"k8s.io/apiserver/pkg/storage/names"

	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	ctrlmetrics "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/metrics"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/patch"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/gitclient"
	lc "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/logchecker"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	orchLibMetrics "github.com/open-edge-platform/orch-library/go/pkg/metrics"
)

const (
	typeReady           = "Ready"
	typeGitSynced       = "GitSynced"
	typeGitReposUpdated = "GitReposUpdated"
	typeNotStalled      = "NotStalled"

	reasonSuccess                 = "Success"
	reasonFailed                  = "Failed"
	reasonInitializing            = "Initializing"
	reasonReconciling             = "Reconciling"
	reasonMockDataLoadFailed      = "MockDataLoadFailed"
	reasonFleetConfigFailed       = "FleetConfigFailed"
	reasonNewGitClientFailed      = "NewGitClientFailed"
	reasonGitRemoteCheckFailed    = "GitRemoteCheckFailed"
	reasonGitInitializationFailed = "GitInitializationFailed"
	reasonGitCloneFailed          = "GitCloneFailed"
	reasonGitCommitFailed         = "GitCommitFailed"
	reasonGitPushFailed           = "GitPushFailed"
	reasonGitRepoUpdateFailed     = "GitRepoUpdateFailed"

	maxErrorBackoff      = 5 * time.Minute
	forceResyncInterval  = time.Minute
	readyWait            = 10 // Seconds
	noTargetClustersWait = 5 * time.Minute
)

var (
	Clock clock.Clock = clock.RealClock{}

	ownerKey               = ".metadata.controller"
	jobOwnerKey            = ".metadata.jobowner"
	apiGVStr               = v1beta1.GroupVersion.String()
	gitjobGVStr            = "gitjob.cattle.io/v1"
	getInclusterConfigFunc = rest.InClusterConfig
	logchecker             *lc.LogChecker
)

// Reconciler reconciles a Deployment object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme

	gitclient               gitclient.ClientCreator
	catalogclient           catalogclient.CatalogClient
	vaultAuthClient         auth.VaultAuth
	deleteGitRepo           bool
	requeueStatus           bool
	fleetGitPollingInterval *metav1.Duration
	recorder                record.EventRecorder
	nexusclient             nexus.Interface
}

// +kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=deploymentclusters,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=fleet.cattle.io,resources=gitrepos,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=fleet.cattle.io,resources=bundles,verbs=get;list;watch
// +kubebuilder:rbac:groups=fleet.cattle.io,resources=bundledeployments,verbs=get;list;watch;delete

func gitRepoIdxFunc(rawObj client.Object) []string {
	gitrepo := rawObj.(*fleetv1alpha1.GitRepo)
	owner := metav1.GetControllerOf(gitrepo)
	if owner == nil || owner.APIVersion != apiGVStr || owner.Kind != "Deployment" {
		return nil
	}
	return []string{owner.Name}
}

func jobIdxFunc(rawObj client.Object) []string {
	job := rawObj.(*batchv1.Job)
	owner := metav1.GetControllerOf(job)
	if owner == nil || owner.APIVersion != gitjobGVStr || owner.Kind != "GitJob" {
		return nil
	}
	return []string{owner.Name}
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) (err error) {
	logchecker = lc.New()

	// Hide internal implementation details in logs by filtering based on regexp
	// and converting to canned strings. External documentation should provide
	// information to users based on error codes.

	// Add more checks here

	r.gitclient = gitclient.NewGitClient

	r.catalogclient, err = catalogclient.NewCatalogClient()
	if err != nil {
		return err
	}

	// M2M auth client
	r.vaultAuthClient, err = auth.NewVaultAuth(utils.GetKeycloakServiceEndpoint(), utils.GetSecretServiceEndpoint(), utils.GetServiceAccount())
	if err != nil {
		return err
	}

	deleteRepo, ok := os.LookupEnv("GITEA_DELETE_REPO_ON_TERMINATE")
	if !ok || deleteRepo == "true" {
		r.deleteGitRepo = true
	}

	pollingInterval, err := utils.GetFleetGitPollingInterval()
	if err != nil {
		return err
	}

	r.fleetGitPollingInterval = pollingInterval

	r.recorder = mgr.GetEventRecorderFor("controller")

	// Add field indexer for .metadata.controller
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &fleetv1alpha1.GitRepo{}, ownerKey, gitRepoIdxFunc); err != nil {
		return err
	}

	// Add field indexer for .metadata.jobowner
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &batchv1.Job{}, jobOwnerKey, jobIdxFunc); err != nil {
		return err
	}

	_, err = ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewTypedWithMaxWaitRateLimiter(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request](), maxErrorBackoff),
		}).
		For(&v1beta1.Deployment{}).
		Owns(&fleetv1alpha1.GitRepo{}).
		Watches(&v1beta1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(r.findDeploymentsForCluster)).
		Build(r)
	if err != nil {
		return fmt.Errorf("failed to setup deployment controller (%v)", err)
	}

	cfg, err := getInclusterConfigFunc()
	if err != nil {
		return fmt.Errorf("failed to get in-cluster config: %v", err)
	}

	r.nexusclient, err = nexus.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create nexus client: %v", err)
	}

	return nil
}

// findDeploymentsForCluster returns a list of Deployment reconcile requests when a cluster is deleted
func (r *Reconciler) findDeploymentsForCluster(ctx context.Context, cluster client.Object) []reconcile.Request {
	log := log.FromContext(ctx)
	
	c := cluster.(*v1beta1.Cluster)
	
	// Only handle cluster deletion events
	if c.DeletionTimestamp == nil {
		return []reconcile.Request{}
	}
	
	log.Info("Cluster deleted, checking for deployments to clean up", "cluster", c.Name)
	
	// Find all deployments that might be affected by this cluster deletion
	deploymentList := &v1beta1.DeploymentList{}
	if err := r.List(ctx, deploymentList); err != nil {
		log.Error(err, "Failed to list deployments when cluster was deleted", "cluster", c.Name)
		return []reconcile.Request{}
	}
	
	var requests []reconcile.Request
	
	for _, deployment := range deploymentList.Items {
		// Check if this deployment targets the deleted cluster by looking for DeploymentClusters
		dcList := &v1beta1.DeploymentClusterList{}
		labels := map[string]string{
			string(v1beta1.DeploymentID): deployment.GetId(),
		}
		if err := r.List(ctx, dcList, client.MatchingLabels(labels)); err != nil {
			log.Error(err, "Failed to list deployment clusters", "deployment", deployment.Name)
			continue
		}
		
		// Check if this deployment targets the deleted cluster
		targetsDeletedCluster := false
		for _, dc := range dcList.Items {
			if dc.Spec.ClusterID == c.Name {
				targetsDeletedCluster = true
				break
			}
		}
		
		if targetsDeletedCluster {
			log.Info("Queueing deployment for reconciliation after cluster deletion", 
				"deployment", deployment.Name, "cluster", c.Name, "type", deployment.Spec.DeploymentType)
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      deployment.Name,
					Namespace: deployment.Namespace,
				},
			})
		}
	}
	
	return requests
}

// shouldDeleteManualDeployment checks if a manual deployment should be deleted
// because all its target clusters have been removed
func (r *Reconciler) shouldDeleteManualDeployment(ctx context.Context, d *v1beta1.Deployment) (bool, error) {
	log := log.FromContext(ctx)
	
	// Only check for targeted (manual) deployments
	if d.Spec.DeploymentType != v1beta1.Targeted {
		return false, nil
	}
	
	// Only consider deployments that are in a steady state (not being created/updated)
	if d.Status.State != v1beta1.Running && d.Status.State != v1beta1.NoTargetClusters {
		return false, nil
	}
	
	// Get all DeploymentClusters for this deployment
	dcList := &v1beta1.DeploymentClusterList{}
	labels := map[string]string{
		string(v1beta1.DeploymentID): d.GetId(),
	}
	if err := r.List(ctx, dcList, client.MatchingLabels(labels)); err != nil {
		return false, fmt.Errorf("failed to list deployment clusters: %v", err)
	}
	
	// If there are no DeploymentClusters and the deployment is not in NoTargetClusters state,
	// this might be a transient condition during creation - don't delete
	if len(dcList.Items) == 0 {
		if d.Status.State != v1beta1.NoTargetClusters {
			log.V(1).Info("Manual deployment has no DeploymentClusters but is not in NoTargetClusters state, skipping deletion check", 
				"deployment", d.Name, "state", d.Status.State)
			return false, nil
		}
		// If deployment is in NoTargetClusters state and has no DeploymentClusters, it's safe to delete
		log.Info("Manual deployment in NoTargetClusters state with no DeploymentClusters, marking for deletion", "deployment", d.Name)
		return true, nil
	}
	
	// Check if any of the target clusters still exist
	hasValidCluster := false
	for _, dc := range dcList.Items {
		cluster := &v1beta1.Cluster{}
		// Clusters are cluster-scoped resources, so no namespace needed
		key := types.NamespacedName{
			Name: dc.Spec.ClusterID,
		}
		
		if err := r.Get(ctx, key, cluster); err == nil {
			// Cluster still exists, don't delete the deployment
			hasValidCluster = true
			break
		} else if !apierrors.IsNotFound(err) {
			// Error other than "not found"
			log.Error(err, "Error checking cluster existence", "cluster", dc.Spec.ClusterID)
			return false, err
		}
		// Cluster not found, continue checking other clusters
	}
	
	// If no valid clusters found and deployment is in NoTargetClusters state, delete it
	if !hasValidCluster && d.Status.State == v1beta1.NoTargetClusters {
		log.Info("All target clusters for manual deployment have been deleted and deployment is in NoTargetClusters state, marking for deletion", 
			"deployment", d.Name, "targetClusters", len(dcList.Items))
		return true, nil
	}
	
	return false, nil
}

// cleanupStaleGitRepos cleans up GitRepo objects that reference deleted clusters
func (r *Reconciler) cleanupStaleGitRepos(ctx context.Context, d *v1beta1.Deployment) error {
	log := log.FromContext(ctx)
	
	// Get all GitRepos owned by this deployment
	gitRepos := &fleetv1alpha1.GitRepoList{}
	if err := r.List(ctx, gitRepos, client.InNamespace(d.Namespace), client.MatchingFields{ownerKey: d.Name}); err != nil {
		return fmt.Errorf("failed to list GitRepos: %v", err)
	}
	
	for i := range gitRepos.Items {
		gr := &gitRepos.Items[i]
		
		// Check if this GitRepo targets any cluster that still exists
		if gr.Spec.Targets != nil && len(gr.Spec.Targets) > 0 {
			hasValidTarget := false
			
			for _, target := range gr.Spec.Targets {
				if target.ClusterSelector != nil && target.ClusterSelector.MatchLabels != nil {
					// Check if any cluster matches this selector
					clusters := &v1beta1.ClusterList{}
					if err := r.List(ctx, clusters, client.MatchingLabels(target.ClusterSelector.MatchLabels)); err != nil {
						log.Error(err, "Failed to list clusters for GitRepo target validation")
						continue
					}
					
					if len(clusters.Items) > 0 {
						hasValidTarget = true
						break
					}
				}
			}
			
			// If this GitRepo has no valid targets, clean up related BundleDeployments
			if !hasValidTarget {
				log.Info("GitRepo has no valid cluster targets, cleaning up related BundleDeployments", 
					"gitrepo", gr.Name)
				
				// Find and clean up BundleDeployments created by this GitRepo
				bdList := &fleetv1alpha1.BundleDeploymentList{}
				bdLabels := map[string]string{
					"fleet.cattle.io/repo-name": gr.Name,
				}
				if err := r.List(ctx, bdList, client.MatchingLabels(bdLabels)); err != nil {
					log.Error(err, "Failed to list BundleDeployments for cleanup", "gitrepo", gr.Name)
					continue
				}
				
				for j := range bdList.Items {
					bd := &bdList.Items[j]
					log.Info("Cleaning up stale BundleDeployment", 
						"bundleDeployment", bd.Name, "gitRepo", gr.Name)
					
					if err := r.Client.Delete(ctx, bd); err != nil && !apierrors.IsNotFound(err) {
						log.Error(err, "Failed to delete stale BundleDeployment", "bundleDeployment", bd.Name)
					}
				}
			}
		}
	}
	
	return nil
}

// cleanupAllDeploymentClusterMetrics aggressively cleans DeploymentCluster metrics
// This prevents timing race conditions where DC controllers might recreate metrics during deletion
func (r *Reconciler) cleanupAllDeploymentClusterMetrics(ctx context.Context, d *v1beta1.Deployment) error {
	log := log.FromContext(ctx)
	
	log.Info("Starting aggressive DeploymentCluster metrics cleanup", 
		"deploymentID", d.GetId(), 
		"deploymentName", d.Name)
	
	// Find all DeploymentClusters that belong to this deployment
	dcList := &v1beta1.DeploymentClusterList{}
	labels := map[string]string{
		string(v1beta1.DeploymentID): d.GetId(),
	}
	
	if err := r.List(ctx, dcList, client.MatchingLabels(labels)); err != nil {
		log.Error(err, "Failed to list DeploymentClusters for metrics cleanup", "deploymentID", d.GetId())
		return err
	}
	
	projectID := ""
	if pid, ok := d.Labels[string(v1beta1.AppOrchActiveProjectID)]; ok {
		projectID = pid
	}
	
	displayName := d.Spec.DisplayName
	
	// Clean metrics for each DeploymentCluster directly using DeploymentCluster metrics API
	for i := range dcList.Items {
		dc := &dcList.Items[i]
		
		log.Info("Cleaning DeploymentCluster metrics directly", 
			"deploymentCluster", dc.Name,
			"clusterID", dc.Spec.ClusterID,
			"deploymentID", d.GetId())
		
		// Clean all possible DC metric states
		states := []v1beta1.StateType{
			v1beta1.Running, v1beta1.Down, v1beta1.Unknown,
		}
		
		for _, state := range states {
			ctrlmetrics.DeploymentClusterStatus.DeleteLabelValues(
				projectID, 
				d.GetId(), 
				displayName, 
				dc.Spec.ClusterID, 
				dc.Status.Name, 
				string(state))
		}
	}
	
	log.Info("Completed aggressive DeploymentCluster metrics cleanup", 
		"cleaned", len(dcList.Items), 
		"deploymentID", d.GetId())
	
	return nil
}

// cleanupOrphanedBundleDeployments cleans up BundleDeployments that reference this deployment
// but may have been left behind when GitRepos were deleted or became stale
func (r *Reconciler) cleanupOrphanedBundleDeployments(ctx context.Context, d *v1beta1.Deployment) error {
	log := log.FromContext(ctx)
	
	// Get all GitRepos owned by this deployment first - we'll need this for multiple strategies
	gitRepos := &fleetv1alpha1.GitRepoList{}
	if err := r.List(ctx, gitRepos, client.InNamespace(d.Namespace), client.MatchingFields{ownerKey: d.Name}); err != nil {
		log.Error(err, "Failed to list GitRepos for BundleDeployment cleanup")
		gitRepos = &fleetv1alpha1.GitRepoList{} // Initialize empty list to prevent nil pointer
	}
	
	log.Info("Starting comprehensive BundleDeployment cleanup", 
		"deploymentID", d.GetId(), 
		"deploymentName", d.Name, 
		"namespace", d.Namespace,
		"ownedGitRepos", len(gitRepos.Items))
	
	// Strategy 1: Find BundleDeployments with deployment ID label (across all namespaces)
	bdList := &fleetv1alpha1.BundleDeploymentList{}
	bdLabels := map[string]string{
		string(v1beta1.DeploymentID): d.GetId(),
	}
	
	if err := r.List(ctx, bdList, client.MatchingLabels(bdLabels)); err != nil {
		log.Error(err, "Failed to list BundleDeployments for orphaned cleanup (strategy 1)")
	} else if len(bdList.Items) > 0 {
		log.Info("Found BundleDeployments to clean up (strategy 1: by deploymentID)", "count", len(bdList.Items), "deploymentID", d.GetId())
		
		for i := range bdList.Items {
			bd := &bdList.Items[i]
			log.Info("Cleaning up orphaned BundleDeployment", 
				"bundleDeployment", bd.Name, "namespace", bd.Namespace, "deploymentID", d.GetId(), "strategy", "deploymentID")
			
			if err := r.Client.Delete(ctx, bd); err != nil && !apierrors.IsNotFound(err) {
				log.Error(err, "Failed to delete orphaned BundleDeployment", "bundleDeployment", bd.Name)
			}
		}
	}
	
	// Strategy 2: Find BundleDeployments via GitRepos owned by this deployment (across all namespaces)
	if len(gitRepos.Items) > 0 {
		log.Info("Processing GitRepos for BundleDeployment cleanup (strategy 2)", "gitRepoCount", len(gitRepos.Items))
		
		for i := range gitRepos.Items {
			gr := &gitRepos.Items[i]
			
			// Find BundleDeployments created by this GitRepo (search across all namespaces)
			bdList2 := &fleetv1alpha1.BundleDeploymentList{}
			bdLabels2 := map[string]string{
				"fleet.cattle.io/repo-name": gr.Name,
			}
			if err := r.List(ctx, bdList2, client.MatchingLabels(bdLabels2)); err != nil {
				log.Error(err, "Failed to list BundleDeployments by repo-name", "gitrepo", gr.Name)
				continue
			}
			
			if len(bdList2.Items) > 0 {
				log.Info("Found BundleDeployments to clean up (strategy 2: by repo-name)", "count", len(bdList2.Items), "gitrepo", gr.Name)
				
				for j := range bdList2.Items {
					bd := &bdList2.Items[j]
					log.Info("Cleaning up orphaned BundleDeployment", 
						"bundleDeployment", bd.Name, "namespace", bd.Namespace, "gitRepo", gr.Name, "strategy", "repo-name")
					
					if err := r.Client.Delete(ctx, bd); err != nil && !apierrors.IsNotFound(err) {
						log.Error(err, "Failed to delete orphaned BundleDeployment", "bundleDeployment", bd.Name)
					}
				}
			}
		}
	} else {
		log.Info("No GitRepos found owned by this deployment for strategy 2", "deploymentName", d.Name)
	}
	
	// Strategy 3: Comprehensive global search for any BundleDeployments that might reference this deployment
	// in their labels, annotations, or through bundle relationships (most comprehensive)
	allBdList := &fleetv1alpha1.BundleDeploymentList{}
	if err := r.List(ctx, allBdList); err != nil {
		log.Error(err, "Failed to list all BundleDeployments for comprehensive cleanup (strategy 3)")
	} else {
		log.Info("Running comprehensive global search (strategy 3)", "totalBundleDeployments", len(allBdList.Items))
		foundCount := 0
		
		for i := range allBdList.Items {
			bd := &allBdList.Items[i]
			
			// Check if this BundleDeployment references our deployment in any way
			shouldDelete := false
			deleteReason := ""
			
			// Check labels
			if bd.Labels != nil {
				if bd.Labels[string(v1beta1.DeploymentID)] == d.GetId() {
					shouldDelete = true
					deleteReason = "deploymentID-label"
				}
				
				// Check if it references any of our GitRepos
				if !shouldDelete {
					if repoName, exists := bd.Labels["fleet.cattle.io/repo-name"]; exists {
						for j := range gitRepos.Items {
							if gitRepos.Items[j].Name == repoName {
								shouldDelete = true
								deleteReason = "gitrepo-reference:" + repoName
								break
							}
						}
					}
				}
				
				// Check for Bundle reference that might link to our deployment
				if !shouldDelete {
					if bundleName, exists := bd.Labels["fleet.cattle.io/bundle-name"]; exists {
						// Check if this bundle belongs to any of our GitRepos by matching naming patterns
						for j := range gitRepos.Items {
							gitRepoName := gitRepos.Items[j].Name
							// Fleet typically creates bundles with names like "gitrepo-name" or "gitrepo-name-<hash>"
							if bundleName == gitRepoName || strings.HasPrefix(bundleName, gitRepoName+"-") {
								shouldDelete = true
								deleteReason = "bundle-pattern-match:" + bundleName
								break
							}
						}
					}
				}
			}
			
			// Check annotations for deployment references
			if !shouldDelete && bd.Annotations != nil {
				if bd.Annotations[string(v1beta1.DeploymentID)] == d.GetId() {
					shouldDelete = true
					deleteReason = "deploymentID-annotation"
				}
			}
			
			// Check if BundleDeployment name contains deployment info (common naming pattern)
			if !shouldDelete {
				deploymentShortID := d.GetId()
				if len(deploymentShortID) > 8 {
					deploymentShortID = deploymentShortID[:8] // Use first 8 chars for pattern matching
				}
				if strings.Contains(bd.Name, deploymentShortID) || strings.Contains(bd.Name, d.Name) {
					shouldDelete = true
					deleteReason = "name-pattern-match"
				}
			}
			
			if shouldDelete {
				foundCount++
				log.Info("Cleaning up orphaned BundleDeployment", 
					"bundleDeployment", bd.Name, 
					"namespace", bd.Namespace, 
					"deploymentID", d.GetId(), 
					"strategy", "comprehensive-search",
					"reason", deleteReason)
				
				if err := r.Client.Delete(ctx, bd); err != nil && !apierrors.IsNotFound(err) {
					log.Error(err, "Failed to delete orphaned BundleDeployment", "bundleDeployment", bd.Name)
				}
			}
		}
		if foundCount > 0 {
			log.Info("Found additional BundleDeployments via comprehensive search", "count", foundCount, "deploymentID", d.GetId())
		} else {
			log.Info("No additional BundleDeployments found in comprehensive search", "deploymentID", d.GetId())
		}
	}
	
	log.Info("Completed comprehensive BundleDeployment cleanup", "deploymentID", d.GetId())
	return nil
}

// cleanupOrphanedDeploymentClusters cleans up DeploymentCluster resources that belong to this deployment
// This is CRITICAL to prevent persistent metrics that cause DeploymentInstanceStatusDown alerts
func (r *Reconciler) cleanupOrphanedDeploymentClusters(ctx context.Context, d *v1beta1.Deployment) error {
	log := log.FromContext(ctx)
	
	log.Info("Starting DeploymentCluster cleanup to prevent metric alerts", 
		"deploymentID", d.GetId(), 
		"deploymentName", d.Name)
	
	// Find all DeploymentClusters that belong to this deployment
	dcList := &v1beta1.DeploymentClusterList{}
	labels := map[string]string{
		string(v1beta1.DeploymentID): d.GetId(),
	}
	
	if err := r.List(ctx, dcList, client.MatchingLabels(labels)); err != nil {
		log.Error(err, "Failed to list DeploymentClusters for cleanup", "deploymentID", d.GetId())
		return err
	}
	
	if len(dcList.Items) == 0 {
		log.Info("No DeploymentClusters found for cleanup", "deploymentID", d.GetId())
		return nil
	}
	
	log.Info("Found DeploymentClusters to clean up", "count", len(dcList.Items), "deploymentID", d.GetId())
	
	// Delete each DeploymentCluster to ensure their metrics are cleaned up
	for i := range dcList.Items {
		dc := &dcList.Items[i]
		
		log.Info("Deleting DeploymentCluster to prevent persistent metrics", 
			"deploymentCluster", dc.Name, 
			"namespace", dc.Namespace,
			"deploymentID", d.GetId(),
			"clusterID", dc.Spec.ClusterID)
		
		if err := r.Client.Delete(ctx, dc); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to delete DeploymentCluster", 
				"deploymentCluster", dc.Name, 
				"deploymentID", d.GetId())
			// Continue with other DeploymentClusters even if one fails
		} else {
			log.Info("Successfully deleted DeploymentCluster", 
				"deploymentCluster", dc.Name,
				"deploymentID", d.GetId())
		}
	}
	
	log.Info("Completed DeploymentCluster cleanup", 
		"deleted", len(dcList.Items), 
		"deploymentID", d.GetId())
	
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrlRes ctrl.Result, reterr error) {
	log := log.FromContext(ctx)

	d := &v1beta1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, d); err != nil {
		// Error reading the object, requeue the request, unless error is "not found"
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create patchHelper and record the current state of the deployment object
	patchHelper, err := patch.NewPatchHelper(d, r.Client)
	if err != nil {
		log.Error(err, "error creating patch helper")
		return ctrl.Result{}, err
	}

	defer func() {
		// CRITICAL FIX: During deletion, only do essential operations to avoid metrics recreation
		if !d.ObjectMeta.DeletionTimestamp.IsZero() {
			log.Info("Deployment being deleted - cleaning metrics and doing essential operations only",
				"deploymentID", d.GetId(),
				"deploymentName", d.Name,
				"finalizers", len(d.ObjectMeta.Finalizers))
			
			// CRITICAL: Always clean metrics during deletion regardless of finalizer state
			// This prevents persistent alerts from any source
			updateStatusMetrics(d, true)
			
			// ESSENTIAL: Still need to patch for finalizer updates during deletion
			if err := patchHelper.Patch(ctx, d); err != nil {
				reterr = kerrors.NewAggregate([]error{reterr, err})
			}
			
			return
		}

		// Normal reconcile path: Always update the status after each reconciliation
		if err := r.updateStatus(ctx, d); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}

		// Patch the updates after each reconciliation
		if err := patchHelper.Patch(ctx, d); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}

		// CRITICAL FIX: Only update metrics for active deployments, never during deletion
		// This prevents the race condition where delete() removes metrics but defer recreates them
		updateStatusMetrics(d, false) // Normal metrics update for active deployments

		// Requeue a reconcile loop to get deployment status since DC wasn't quite
		// ready after readyWait retrigger right after readyWait secs
		if r.requeueStatus {
			log.Info("requeue a reconcile loop to get deployment status since DC wasn't quite ready")
			ctrlRes.RequeueAfter = readyWait * time.Second
		}
	}()

	// Add finalizer first to avoid the race condition between init and delete.
	// If webhook is enabled, webhook sets the finalizers to avoid extra
	// reconciliation loop.
	if r.deleteGitRepo && !cutil.ContainsFinalizer(d, v1beta1.FinalizerGitRemote) && d.ObjectMeta.DeletionTimestamp.IsZero() {
		cutil.AddFinalizer(d, v1beta1.FinalizerGitRemote)
		return ctrl.Result{}, nil
	}

	// Add finalizer to avoid race condition
	// If webhook is enabled, webhook sets the finalizers to avoid extra
	// reconciliation loop.
	if !cutil.ContainsFinalizer(d, v1beta1.FinalizerDependency) && d.ObjectMeta.DeletionTimestamp.IsZero() {
		cutil.AddFinalizer(d, v1beta1.FinalizerDependency)
		return ctrl.Result{}, nil
	}

	// Handle finalizers if the deletion timestamp is non-zero
	if !d.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.delete(ctx, d)
	}

	// Always clean up any orphaned BundleDeployments during reconciliation
	// This ensures that stale BundleDeployments are removed regardless of deployment state
	if err := r.cleanupOrphanedBundleDeployments(ctx, d); err != nil {
		log.Error(err, "Failed to cleanup orphaned BundleDeployments", "deployment", d.Name)
		// Don't fail the reconciliation, just log the error and continue
	}

	// Only check for cleanup if deployment is in NoTargetClusters state or error state
	// This prevents premature deletion during normal reconciliation
	if d.Status.State == v1beta1.NoTargetClusters || d.Status.State == v1beta1.Error {
		// Check if this manual deployment should be deleted because all target clusters are gone
		if shouldDelete, err := r.shouldDeleteManualDeployment(ctx, d); err != nil {
			log.Error(err, "Failed to check if manual deployment should be deleted", "deployment", d.Name)
			return ctrl.Result{}, err
		} else if shouldDelete {
			log.Info("Deleting manual deployment because all target clusters have been removed", "deployment", d.Name)
			if err := r.Delete(ctx, d); err != nil && !apierrors.IsNotFound(err) {
				log.Error(err, "Failed to delete manual deployment", "deployment", d.Name)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		// Clean up any stale GitRepo objects that reference deleted clusters
		if err := r.cleanupStaleGitRepos(ctx, d); err != nil {
			log.Error(err, "Failed to cleanup stale GitRepos", "deployment", d.Name)
			// Don't fail the reconciliation, just log the error and continue
		}
	}

	// If no changes to Deployment Spec since the last successful reconcile,
	// force redeploy stucked apps and skip normal reconciliation loops
	ready := meta.IsStatusConditionTrue(d.Status.Conditions, typeReady)
	changed, err := r.gitURLHasChanged(ctx, d)
	if err != nil {
		return ctrl.Result{}, err
	}
	if ready && (!changed) && d.Status.ReconciledGeneration == d.Generation {
		// Since Fleet v0.7.0, explicit force update is required if
		// Deployment update gets stuck on an error (LPOD-2953)
		if d.Status.DeployInProgress {
			return ctrl.Result{}, r.forceRedeployStuckApps(ctx, d)
		}
		return ctrl.Result{}, nil
	}

	// New Deployment or updates to the existing Deployment Spec has been
	// detected. Proceed with the normal reconciliation loops.
	result, err := r.reconcile(ctx, d)
	if err != nil {
		r.recorder.Eventf(d, corev1.EventTypeWarning, "ReconcileError", "Reconcile failed: %v", err)
	}

	return result, err
}

func (r *Reconciler) delete(ctx context.Context, d *v1beta1.Deployment) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	
	// CRITICAL FIX: Clean up ALL metrics IMMEDIATELY when deletion starts
	// This prevents persistent Prometheus metrics that cause DeploymentInstanceStatusDown alerts
	log.Info("Cleaning up all Deployment and DeploymentCluster metrics immediately during deletion", 
		"deploymentID", d.GetId(), 
		"deploymentName", d.Name)
	
	// Clean Deployment metrics first
	updateStatusMetrics(d, true) // Force delete metrics regardless of finalizer state
	
	// CRITICAL: Also clean DeploymentCluster metrics directly to prevent timing race conditions
	if err := r.cleanupAllDeploymentClusterMetrics(ctx, d); err != nil {
		log.Error(err, "Failed to cleanup DeploymentCluster metrics during deletion", "deployment", d.Name)
		// Continue - this is not blocking but critical for alert cleanup
	}
	
	// CRITICAL FIX: Clean up DeploymentClusters SECOND to prevent persistent metrics
	// This is another root cause of DeploymentInstanceStatusDown alerts persisting after deletion
	if err := r.cleanupOrphanedDeploymentClusters(ctx, d); err != nil {
		log.Error(err, "Failed to cleanup DeploymentClusters during deletion", "deployment", d.Name)
		// Continue with deletion even if this fails - metrics cleanup is critical but not blocking
	}
	
	// Third, aggressively clean up all orphaned BundleDeployments during deletion
	// This ensures Fleet references are cleaned up immediately
	if err := r.cleanupOrphanedBundleDeployments(ctx, d); err != nil {
		log.Error(err, "Failed to cleanup orphaned BundleDeployments during deletion", "deployment", d.Name)
		// Continue with deletion even if this fails
	}
	
	phases := []func(context.Context, *v1beta1.Deployment) (ctrl.Result, error){
		r.handleFinalizerDependency,
		r.handleFinalizerGitRemote,
		r.handleFinalizerCatalog,
	}

	errs := []error{}
	for _, phase := range phases {
		if _, err := phase(ctx, d); err != nil {
			errs = append(errs, err)
		}
	}

	// FINAL SAFETY: One more aggressive metrics cleanup after finalizer phases
	// This ensures any lingering metrics are cleaned up before deletion completes
	log.Info("Final aggressive metrics cleanup after finalizer phases", 
		"deploymentID", d.GetId(),
		"deploymentName", d.Name,
		"finalizers", len(d.ObjectMeta.Finalizers))
		
	updateStatusMetrics(d, true) // Clean Deployment metrics again
	if err := r.cleanupAllDeploymentClusterMetrics(ctx, d); err != nil {
		log.Error(err, "Failed final DeploymentCluster metrics cleanup", "deployment", d.Name)
		// Don't add to errs - this is safety cleanup, not critical for deletion
	}

	return ctrl.Result{}, kerrors.NewAggregate(errs)
}

func (r *Reconciler) reconcile(ctx context.Context, d *v1beta1.Deployment) (ctrl.Result, error) {
	phases := []func(context.Context, *v1beta1.Deployment) (ctrl.Result, error){
		r.reconcileState,
		r.reconcileDependency,
		r.reconcileRepository,
		r.reconcileGitRepo,
	}

	errs := []error{}
	for _, phase := range phases {
		if _, err := phase(ctx, d); err != nil {
			errs = append(errs, err)
		}
	}

	// Update ReconciledGeneration to Generation when no errors
	if len(errs) == 0 {
		d.Status.ReconciledGeneration = d.Generation
		d.Status.Conditions = utils.UpdateStatusCondition(d.Status.Conditions, typeReady, metav1.ConditionTrue, reasonSuccess, nil)
	} else {
		d.Status.Conditions = utils.UpdateStatusCondition(d.Status.Conditions, typeReady, metav1.ConditionFalse, reasonFailed, nil)
	}

	return ctrl.Result{}, kerrors.NewAggregate(errs)
}

func (r *Reconciler) handleFinalizerDependency(ctx context.Context, d *v1beta1.Deployment) (ctrl.Result, error) {
	if !cutil.ContainsFinalizer(d, v1beta1.FinalizerDependency) {
		return ctrl.Result{}, nil
	}

	log := log.FromContext(ctx)

	// skip if there is no children
	if len(d.Spec.ChildDeploymentList) == 0 {
		cutil.RemoveFinalizer(d, v1beta1.FinalizerDependency)
		log.V(2).Info("Removing finalizer", "finalizer", v1beta1.FinalizerDependency)
		return ctrl.Result{}, nil
	}

	// for the scenario to delete parent deployment
	for k := range d.Spec.ChildDeploymentList {
		cd := &v1beta1.Deployment{}

		// get child deployment CR
		if err := r.Get(ctx, types.NamespacedName{Namespace: d.Namespace, Name: k}, cd); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "error retrieving child deployment")
				return ctrl.Result{}, fmt.Errorf("children is not created yet - will requeue: %+v", err)
			}
			log.Info("cannot find child deployment; probably child deployment is already removed")
			continue
		}

		// define patchHelper to update child deployment CR
		childPatchHelper, err := patch.NewPatchHelper(cd, r.Client)
		if err != nil {
			log.Error(err, "error creating patch helper")
			return ctrl.Result{}, err
		}

		// if parentDeploymentList is empty, create a map
		if len(cd.Status.ParentDeploymentList) == 0 {
			continue
		}

		// remove parent info from child deployment
		delete(cd.Status.ParentDeploymentList, d.Name)

		// update child status
		if err := childPatchHelper.Patch(ctx, cd); err != nil {
			log.Error(err, "error patching child status")
			return ctrl.Result{}, fmt.Errorf("children status could not be updated - will retry: %+v", err)
		}
	}

	cutil.RemoveFinalizer(d, v1beta1.FinalizerDependency)
	log.V(2).Info("Removing finalizer", "finalizer", v1beta1.FinalizerDependency)
	return ctrl.Result{}, nil
}

func (r *Reconciler) handleFinalizerGitRemote(ctx context.Context, d *v1beta1.Deployment) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !cutil.ContainsFinalizer(d, v1beta1.FinalizerGitRemote) {
		return ctrl.Result{}, nil
	}

	repository, err := r.gitclient(d.GetId())
	if err == nil {
		if exists, err := repository.ExistsOnRemote(); err == nil && exists {
			if err = repository.Delete(); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		return ctrl.Result{}, err
	}

	cutil.RemoveFinalizer(d, v1beta1.FinalizerGitRemote)
	log.V(2).Info("Removing finalizer", "finalizer", v1beta1.FinalizerGitRemote)

	return ctrl.Result{}, nil
}

func (r *Reconciler) handleFinalizerCatalog(ctx context.Context, d *v1beta1.Deployment) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !cutil.ContainsFinalizer(d, v1beta1.FinalizerCatalog) {
		return ctrl.Result{}, nil
	}

	isDeployed, err := utils.IsDeployed(ctx, r.Client, d)
	if err == nil && !isDeployed {
		err = utils.HandleIsDeployed(ctx, r.catalogclient, r.vaultAuthClient, d, false)
	}

	if err != nil {
		if strings.Contains(err.Error(), "failed to unset isDeployed") && strings.Contains(err.Error(), "rpc error: code = NotFound") {
			log.V(2).Info("Failed to retrieve the deployment package from catalog; normally deleted by Catalog when deleting project - delete deployment anyway", "CatalogErr", err)
			cutil.RemoveFinalizer(d, v1beta1.FinalizerCatalog)
			log.V(2).Info("Removing finalizer", "finalizer", v1beta1.FinalizerCatalog)
		} else {
			log.Error(err, "failed for processing catalog finalizer")
			return ctrl.Result{}, err
		}
	} else {
		cutil.RemoveFinalizer(d, v1beta1.FinalizerCatalog)
		log.V(2).Info("Removing finalizer", "finalizer", v1beta1.FinalizerCatalog)
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileDependency(ctx context.Context, d *v1beta1.Deployment) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// skip if there is no children
	if len(d.Spec.ChildDeploymentList) == 0 {
		return ctrl.Result{}, nil
	}

	// for scenarios to create and update parent deployment
	for k := range d.Spec.ChildDeploymentList {
		cd := &v1beta1.Deployment{}

		// get child deployment CR
		if err := r.Get(ctx, types.NamespacedName{Namespace: d.Namespace, Name: k}, cd); err != nil {
			// Error reading the object, requeue the request, unless error is "not found"
			log.Error(err, "error retrieving child deployment")
			return ctrl.Result{}, fmt.Errorf("children is not created yet - will requeue: %+v", err)
		}

		// define patchHelper to update child deployment CR
		childPatchHelper, err := patch.NewPatchHelper(cd, r.Client)
		if err != nil {
			log.Error(err, "error creating patch helper")
			return ctrl.Result{}, err
		}

		// if parentDeploymentList is empty or nil, create a map
		if cd.Status.ParentDeploymentList == nil {
			cd.Status.ParentDeploymentList = make(map[string]v1beta1.DependentDeploymentRef)
		}

		// case 1. if parent info is not in child Deployment parent list, create a new one
		// case 2. if parent info in child Deployment parent list is out-of-date, update a new one
		// case 3. otherwise, skip
		pd, ok := cd.Status.ParentDeploymentList[d.Name]
		if !ok || !reflect.DeepEqual(pd.DeploymentPackageRef, d.Spec.DeploymentPackageRef) || pd.DeploymentName != d.Name {
			// case 1 or 2
			cd.Status.ParentDeploymentList[d.Name] = v1beta1.DependentDeploymentRef{
				DeploymentPackageRef: *d.Spec.DeploymentPackageRef.DeepCopy(),
				DeploymentName:       d.Name,
			}
		} else {
			// case 3
			continue
		}

		// update child status
		if err := childPatchHelper.Patch(ctx, cd); err != nil {
			log.Error(err, "error patching child status")
			return ctrl.Result{}, fmt.Errorf("children status could not be updated - will retry: %+v", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileState(_ context.Context, d *v1beta1.Deployment) (ctrl.Result, error) {
	// Update state to Deploying or Updating
	if d.Status.State == "" || d.Generation == 1 {
		projectID := d.Labels[string(v1beta1.AppOrchActiveProjectID)]
		orchLibMetrics.RecordTimestamp(projectID, d.GetId(), d.Spec.DisplayName, "start", "CreateDeployment")
		d.Status.State = v1beta1.Deploying
	} else {
		d.Status.State = v1beta1.Updating
	}

	// Reset conditions before updating
	for _, c := range d.Status.Conditions {
		meta.RemoveStatusCondition(&d.Status.Conditions, c.Type)
	}

	d.Status.DeployInProgress = true
	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileRepository(_ context.Context, d *v1beta1.Deployment) (result ctrl.Result, err error) {
	status := metav1.ConditionFalse
	reason := ""

	// Update the GitSynced condition before return
	defer func() {
		d.Status.Conditions = utils.UpdateStatusCondition(d.Status.Conditions, typeGitSynced, status, reason, err)
	}()

	var gc gitclient.Repository
	var repoExists bool

	// Clean up local repo path if exists
	basedir := filepath.Join("/tmp", d.GetId())
	os.RemoveAll(basedir)

	if gc, err = r.gitclient(d.GetId()); err != nil {
		reason = reasonNewGitClientFailed
		return ctrl.Result{}, err
	}

	if repoExists, err = gc.ExistsOnRemote(); err != nil {
		reason = reasonGitRemoteCheckFailed
		return ctrl.Result{}, err
	}

	// Clone remote repo to basedir if the repo exists, otherwise initialize the repo from basedir
	if repoExists {
		if err := gc.Clone(basedir); err != nil {
			reason = reasonGitCloneFailed
			return ctrl.Result{}, err
		}
	} else {
		if err := gc.Initialize(basedir); err != nil {
			reason = reasonGitInitializationFailed
			return ctrl.Result{}, err
		}
	}
	// Generate fleet configurations for the applications and commit,
	// and push to the remote git repository
	if err := fleet.GenerateFleetConfigs(d, basedir, r.Client, r.nexusclient.RuntimeprojectEdgeV1()); err != nil {
		reason = reasonFleetConfigFailed
		return ctrl.Result{}, err
	}

	if err := gc.CommitFiles(); err != nil {
		reason = reasonGitCommitFailed
		return ctrl.Result{}, err
	}

	if err := gc.PushToRemote(); err != nil {
		reason = reasonGitPushFailed
		return ctrl.Result{}, err
	}

	// Remote git repository is now synced with the latest Fleet configurations
	// Set the condition to true and record the event with the repo name
	status = metav1.ConditionTrue
	reason = reasonSuccess
	r.recorder.Eventf(d, corev1.EventTypeNormal, "Reconciling", "Completed sync git repository %s", d.GetId())

	return ctrl.Result{}, nil
}

func (r *Reconciler) gitURLHasChanged(ctx context.Context, d *v1beta1.Deployment) (changed bool, err error) {
	log := log.FromContext(ctx)

	gitRepoURL, err := gitclient.GetRemoteURLWithCreds(d.GetId())
	if err != nil {
		return false, err
	}

	// Fetch the Deployment's GitRepos
	var childGitRepos fleetv1alpha1.GitRepoList
	if err := r.List(ctx, &childGitRepos, client.InNamespace(d.Namespace), client.MatchingFields{ownerKey: d.Name}); err != nil {
		return false, err
	}

	for i := range childGitRepos.Items {
		gr := &childGitRepos.Items[i]
		if gr.Spec.Repo != gitRepoURL {
			log.Info("Detected Git repo URL change for Deployment")
			return true, nil
		}
	}

	return false, nil
}

func (r *Reconciler) reconcileGitRepo(ctx context.Context, d *v1beta1.Deployment) (result ctrl.Result, err error) {
	status := metav1.ConditionFalse
	reason := reasonGitRepoUpdateFailed

	// Update GitReposUpdated condition before return
	defer func() {
		d.Status.Conditions = utils.UpdateStatusCondition(d.Status.Conditions, typeGitReposUpdated, status, reason, err)
	}()

	// TODO: remove creds from git repo URL when Fleet v0.9.0 is released
	// Generate remote git repository URL for this Deployment
	gitRepoURL, err := gitclient.GetRemoteURLWithCreds(d.GetId())
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create or update GitRepo objects, one per application to allow the case
	// where each application deployed in a different targets

	// Fetch the Deployment's GitRepos
	var childGitRepos fleetv1alpha1.GitRepoList
	if err := r.List(ctx, &childGitRepos, client.InNamespace(d.Namespace), client.MatchingFields{ownerKey: d.Name}); err != nil {
		return ctrl.Result{}, err
	}

	// Create a map of the GitRepos
	grmap := make(map[string]*fleetv1alpha1.GitRepo, len(childGitRepos.Items))
	for i := range childGitRepos.Items {
		gr := &childGitRepos.Items[i]
		grmap[gr.Name] = gr
	}

	for _, app := range d.Spec.Applications {
		gitRepoName := getGitRepoName(app.Name, d.GetId())
		gitRepoNamespace := d.Namespace
		gitRepoTargets := []fleetv1alpha1.GitTarget{}
		if len(app.Targets) > 0 {
			for _, t := range app.Targets {
				gitRepoTargets = append(gitRepoTargets, fleetv1alpha1.GitTarget{
					Name: names.SimpleNameGenerator.GenerateName("match-"),
					ClusterSelector: &metav1.LabelSelector{
						MatchLabels: t,
					},
				})
			}
		}

		gitRepo, gitRepoExists := grmap[gitRepoName]

		if gitRepoExists {
			// Remove it from the map to show we've processed it
			delete(grmap, gitRepoName)

			//In case of upgrade, gitrepos will be already present.
			//existing gitrepo CRs.
			caCert := utils.GetGitCaCert()
			// Set the CABundle for the GitRepo if one was provided
			if caCert != "" {
				gitRepo.Spec.CABundle = []byte(caCert)
			}
			gitRepo.Spec.ClientSecretName = v1beta1.FleetGitSecretName
			// GitRepo already exists, update the existing one
			gitRepo.ObjectMeta.Labels[string(v1beta1.BundleName)] = fleet.BundleName(app, d.GetName())
			gitRepo.Spec.Repo = gitRepoURL
			gitRepo.Spec.HelmSecretName = app.HelmApp.RepoSecretName
			gitRepo.Spec.Targets = gitRepoTargets

			if err := r.Client.Update(ctx, gitRepo); err != nil {
				return ctrl.Result{}, fmt.Errorf("GitRepo update failed(%v)", err)
			}

			r.recorder.Eventf(d, corev1.EventTypeNormal, "Reconciling", "Completed updating GitRepo %s", gitRepoName)
		} else {
			caCert := utils.GetGitCaCert()
			// GitRepo does not exist, create a new one
			gitRepo = &fleetv1alpha1.GitRepo{ObjectMeta: metav1.ObjectMeta{
				Namespace: gitRepoNamespace,
				Name:      gitRepoName,
				Labels: map[string]string{
					string(v1beta1.BundleName): fleet.BundleName(app, d.GetName()),
					string(v1beta1.BundleType): fleet.BundleTypeApp.String(),
				},
			},
				Spec: fleetv1alpha1.GitRepoSpec{
					Repo:             gitRepoURL,
					Paths:            []string{app.Name},
					HelmSecretName:   app.HelmApp.RepoSecretName,
					ClientSecretName: v1beta1.FleetGitSecretName,
					Targets:          gitRepoTargets,
					PollingInterval:  r.fleetGitPollingInterval,
				},
			}
			// Set the CABundle for the GitRepo if one was provided
			if caCert != "" {
				gitRepo.Spec.CABundle = []byte(caCert)
			}

			if activeProjectID, ok := d.Labels[string(v1beta1.AppOrchActiveProjectID)]; ok {
				gitRepo.Labels[string(v1beta1.AppOrchActiveProjectID)] = activeProjectID
			} else {
				return ctrl.Result{}, fmt.Errorf("GitRepo creation failed (%v)", err)
			}

			if err := ctrl.SetControllerReference(d, gitRepo, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}

			if err := r.Client.Create(ctx, gitRepo); err != nil {
				return ctrl.Result{}, fmt.Errorf("GitRepo creation failed (%v)", err)
			}

			// GitRepo object for the application was successful
			r.recorder.Eventf(d, corev1.EventTypeNormal, "Reconciling", "Completed creating GitRepo %s", gitRepoName)

			projectID := d.Labels[string(v1beta1.AppOrchActiveProjectID)]
			orchLibMetrics.RecordTimestamp(projectID, d.GetId(), d.Spec.DisplayName, "start", "CreateGitRepo")
			orchLibMetrics.CalculateTimeDifference(projectID, d.GetId(), d.Spec.DisplayName, "start", "CreateDeployment", "start", "CreateGitRepo")
		}
	}

	// Delete any GitRepos remaining in the grmap since they don't correspond to existing apps
	for name, gitRepo := range grmap {
		log := log.FromContext(ctx)
		log.Info(fmt.Sprintf("Deleting orphaned GitRepo %s", name))
		if err := r.Client.Delete(ctx, gitRepo); err != nil {
			log.Error(err, "Failed to delete GitRepo")
			return ctrl.Result{}, err
		}
	}

	status = metav1.ConditionTrue
	reason = reasonSuccess
	return ctrl.Result{}, nil
}

func (r *Reconciler) updateStatus(ctx context.Context, d *v1beta1.Deployment) error {
	// If the deployment is deleted, do set the state to terminating and return
	if !d.ObjectMeta.DeletionTimestamp.IsZero() {
		d.Status.State = v1beta1.Terminating
		return nil
	}

	// Fetch the Deployment's GitRepos
	var childGitRepos fleetv1alpha1.GitRepoList
	if err := r.List(ctx, &childGitRepos, client.InNamespace(d.Namespace), client.MatchingFields{ownerKey: d.Name}); err != nil {
		return err
	}

	// Fetch the Deployment's DeploymentClusters
	var deploymentClusters v1beta1.DeploymentClusterList
	labels := map[string]string{
		string(v1beta1.DeploymentID): d.GetId(),
	}

	if err := r.List(ctx, &deploymentClusters, client.MatchingLabels(labels)); err != nil {
		return err
	}

	// Fleet v0.8 does not report errors in the GitJob pod that downloads Git repos and Helm charts
	// See this issue: https://github.com/rancher/fleet/issues/2065
	// Extract error messages from the GitJob pods ourselves and store in GitRepo
	if d.Status.DeployInProgress {
		// store status per gitrepo
		condMapAllGitRepos := make(map[string]*metav1.Condition)

		for i := range childGitRepos.Items {
			gitRepo := &childGitRepos.Items[i]
			if err := r.updateWithGitJobStatus(ctx, d, gitRepo, condMapAllGitRepos); err != nil {
				return err
			}
		}

		// calculate NotStalled condition
		msgs := make([]string, 0)
		allSucceed := true
		for _, v := range condMapAllGitRepos {
			if v.Reason == reasonFailed {
				allSucceed = false
				msgs = append(msgs, v.Message)
			}
		}

		// case 1: one or more job pods are in Failed state
		// case 2: all job pods are in Success (allSucceed == true && #condMapAllGitRepos and #childGitRepos are the same)
		// case 3: Otherwise - if one or some of git job pods are not in either Completed or Error - no need to take any action
		if !allSucceed {
			// case 1
			d.Status.Conditions = utils.UpdateStatusCondition(d.Status.Conditions,
				typeNotStalled,
				metav1.ConditionFalse,
				reasonFailed,
				fmt.Errorf("%+v", msgs))
		} else if len(childGitRepos.Items) == len(condMapAllGitRepos) {
			// case2
			d.Status.Conditions = utils.UpdateStatusCondition(d.Status.Conditions,
				typeNotStalled,
				metav1.ConditionTrue,
				reasonSuccess,
				nil)
		}

	}

	// If there was error during the reconciliation or stalled condition found,
	// set the state to internal error and return
	for _, cond := range d.Status.Conditions {
		if cond.Status == metav1.ConditionFalse {
			d.Status.Message = cond.Message
			d.Status.State = v1beta1.InternalError
			return nil
		}
	}

	r.updateDeploymentStatus(d, childGitRepos.Items, deploymentClusters.Items)
	return nil
}

func (r *Reconciler) forceRedeployStuckApps(ctx context.Context, d *v1beta1.Deployment) error {
	if d.Status.LastForceResync == "" {
		// Never force updated before, just record the current time and return
		d.Status.LastForceResync = time.Now().Format(time.RFC3339)
		return nil
	}

	// Check timestamp since last force update; return if too soon
	lastForceResync, err := time.Parse(time.RFC3339, d.Status.LastForceResync)
	if err != nil {
		return fmt.Errorf("failed to parse LastForceResync (%v)", err)
	}
	if time.Since(lastForceResync) < forceResyncInterval {
		return nil
	}

	// Get GitRepo objects owned by the Deployment
	gitRepos := &fleetv1alpha1.GitRepoList{}
	err = r.List(ctx, gitRepos, client.InNamespace(d.Namespace), client.MatchingFields{ownerKey: d.Name})
	if err != nil {
		return fmt.Errorf("failed list GitRepos (%v)", err)
	}

	// Loop over GitRepo objects owned by the Deployment and check the conditions
	for i := range gitRepos.Items {
		gitRepo := &gitRepos.Items[i]
		forceUpdate := false

		// Check for "Unable to continue" message
		c, ok := utils.GetGenericCondition(&gitRepo.Status.Conditions, "Ready")
		if ok && c.Status == "False" && strings.Contains(c.Message, "Unable to continue") {
			forceUpdate = true
		}

		// Check for Stalled condition
		c, ok = utils.GetGenericCondition(&gitRepo.Status.Conditions, "Stalled")
		if ok && c.Status == "True" {
			forceUpdate = true
		}

		if forceUpdate {
			app := getAppNameForGitRepo(gitRepo, d.GetId())

			gitRepo.Spec.ForceSyncGeneration++
			if err := r.Client.Update(ctx, gitRepo); err != nil {
				return fmt.Errorf("failed to force sync app %s(%v)", app, err)
			}

			d.Status.LastForceResync = time.Now().Format(time.RFC3339)
			r.recorder.Eventf(d, corev1.EventTypeNormal, "Reconciling", "Force sync triggered for app %s", app)
		}
	}
	return nil
}

func (r *Reconciler) updateWithGitJobStatus(ctx context.Context, d *v1beta1.Deployment, gitRepo *fleetv1alpha1.GitRepo, condMapAllGitRepos map[string]*metav1.Condition) error {
	var gitjobPods corev1.PodList
	var gitJobs batchv1.JobList

	// Get Job for this GitRepo
	if err := r.List(ctx, &gitJobs, client.MatchingFields{jobOwnerKey: gitRepo.Name}); err != nil {
		return err
	}

	if len(gitJobs.Items) < 1 {
		// Return early if no job exists yet
		return nil
	}

	job := gitJobs.Items[0]
	if job.Status.Succeeded == 1 {
		condMapAllGitRepos[gitRepo.Name] = &metav1.Condition{
			Type:               typeNotStalled,
			LastTransitionTime: metav1.NewTime(Clock.Now()),
			Status:             metav1.ConditionTrue,
			Reason:             reasonSuccess,
			Message:            utils.MessageFromError(nil),
		}
		return nil
	}

	labels := map[string]string{"job-name": job.Name}
	if err := r.List(ctx, &gitjobPods, client.MatchingLabels(labels)); err != nil {
		return err
	}

	// Pick the latest error state job pod if there is
	var errPod *corev1.Pod
	for _, p := range gitjobPods.Items {
		if p.Status.Phase != "Failed" {
			continue
		}
		if errPod == nil {
			errPod = &p
		} else if errPod.CreationTimestamp.Before(&p.CreationTimestamp) {
			errPod = &p
		}
	}

	// Deployment job is not completed but no error status pod found
	// The job pod is Initializing or NotReady
	if errPod == nil {
		return nil
	}

	// Retrieve error message from either fleet or source-git-repo pods (only one has message)
	// and store it to Deployment Stalled condition
	for _, cs := range errPod.Status.ContainerStatuses {
		if cs.State.Terminated != nil && cs.State.Terminated.Reason == "Error" && cs.State.Terminated.Message != "" {
			condMapAllGitRepos[gitRepo.Name] = &metav1.Condition{
				Type:               typeNotStalled,
				LastTransitionTime: metav1.NewTime(Clock.Now()),
				Status:             metav1.ConditionFalse,
				Reason:             reasonFailed,
				Message:            utils.MessageFromError(fmt.Errorf("%s: %s", getAppNameForGitRepo(gitRepo, d.GetId()), cs.State.Terminated.Message)),
			}
		}
	}
	return nil
}

func updateStatusMetrics(d *v1beta1.Deployment, deleteMetrics bool) {
	metricValue := make(map[string]float64)

	// Init all values to 0
	metricValue[string(v1beta1.Running)] = 0
	metricValue[string(v1beta1.Down)] = 0
	metricValue[string(v1beta1.Unknown)] = 0
	metricValue[string(v1beta1.Error)] = 0
	metricValue[string(v1beta1.InternalError)] = 0
	metricValue[string(v1beta1.Deploying)] = 0
	metricValue[string(v1beta1.Updating)] = 0
	metricValue[string(v1beta1.Terminating)] = 0
	metricValue[string(v1beta1.NoTargetClusters)] = 0

	displayName := d.Spec.DisplayName

	projectID := ""
	if _, ok := d.Labels[string(v1beta1.AppOrchActiveProjectID)]; ok {
		projectID = d.Labels[string(v1beta1.AppOrchActiveProjectID)]
	}

	if deleteMetrics {
		// Delete current deployment metrics only - prevents DeploymentInstanceStatusDown alerts
		for i := range metricValue {
			ctrlmetrics.DeploymentStatus.DeleteLabelValues(projectID, d.GetId(), displayName, i)
		}

		orchLibMetrics.DeleteTimestampMetrics(projectID, d.GetId())
	} else {
		// Only one status will be 1 and rest are 0
		metricValue[string(d.Status.State)] = 1

		// Update and output all metrics
		for i, val := range metricValue {
			ctrlmetrics.DeploymentStatus.WithLabelValues(projectID, d.GetId(), displayName, i).Set(val)
		}

	}
}

func (r *Reconciler) updateDeploymentStatus(d *v1beta1.Deployment, grlist []fleetv1alpha1.GitRepo, dclist []v1beta1.DeploymentCluster) {
	var newState v1beta1.StateType
	stalledApps := false
	apps := 0
	message := ""
	r.requeueStatus = false

	// Walk GitRepos for the Deployment to extract any error conditions
	for i := range grlist {
		gitrepo := grlist[i]
		apps++
		appName := getAppNameForGitRepo(&gitrepo, d.GetId())

		if d.Status.DeployInProgress {

			// Check if the GitRepo is in Stalled state
			if sc, ok := utils.GetGenericCondition(&gitrepo.Status.Conditions, "Stalled"); ok && sc.Status == corev1.ConditionTrue {
				stalledApps = true
				message = utils.AppendMessage(logchecker.ProcessLog(message), fmt.Sprintf("App %s: %s", appName, sc.Message))
			}
		}

		// Record the message if there is one
		if gitrepo.Status.Display.Message != "" {
			message = utils.AppendMessage(logchecker.ProcessLog(message), fmt.Sprintf("App %s: %s", appName, gitrepo.Status.Display.Message))
		}
	}

	// Check deployment ready condition to extract error message
	if d.Status.DeployInProgress {
		cond := meta.FindStatusCondition(d.Status.Conditions, typeNotStalled)
		if cond != nil && cond.Status == metav1.ConditionFalse {
			stalledApps = true
			message = utils.AppendMessage(message, cond.Message)
		}
	}

	// Walk DeploymentClusters for the Deployment and generate cluster counts
	clustercounts := v1beta1.ClusterSummary{
		Total:   len(dclist),
		Running: 0,
		Down:    0,
		Unknown: 0,
	}
	for _, dc := range dclist {
		switch dc.Status.Status.State {
		case v1beta1.Unknown:
			clustercounts.Unknown++
		case v1beta1.Down:
			clustercounts.Down++
			if strings.Contains(dc.Status.Status.Message, "Progress deadline exceeded") {
				stalledApps = true
				message = utils.AppendMessage(message, dc.Status.Status.Message)
			}
		case v1beta1.Running:
			ready := true
			for _, app := range dc.Status.Apps {
				if d.Generation != app.DeploymentGeneration {
					ready = false
				}
			}

			// Wait a few seconds before deciding the DC is actually Ready
			cond := meta.FindStatusCondition(dc.Status.Conditions, typeReady)
			if cond == nil || Clock.Since(cond.LastTransitionTime.Time).Seconds() <= readyWait {
				// Requeue a reconcile loop to get deployment status since DC wasn't quite
				// ready after readyWait retrigger right after readyWait secs
				r.requeueStatus = true
				ready = false
			}

			if ready {
				clustercounts.Running++
			} else {
				clustercounts.Down++
			}
		}
	}

	// Calculate the Deployment's state
	switch {
	case stalledApps:
		newState = v1beta1.Error
	case clustercounts.Unknown > 0:
		newState = v1beta1.Unknown
	case clustercounts.Total == 0:
		// Wait specified interval after creation before showing NoTargetClusters,
		// to give Fleet + ADM a chance to bootstrap the Deployment.
		if time.Now().After(d.CreationTimestamp.Time.Add(noTargetClustersWait)) {
			newState = v1beta1.NoTargetClusters
		} else {
			// If deployment was already running and cluster went down
			// before (d.CreationTimestamp.Time.Add(noTargetClustersWait)) then set NoTargetClusters
			if d.Status.DeployInProgress {
				newState = v1beta1.Deploying
			} else {
				newState = v1beta1.NoTargetClusters
			}
		}

	case clustercounts.Down > 0 || clustercounts.Total > clustercounts.Running:
		// Ignore Down state Deployment is updating
		if d.Status.DeployInProgress {
			if d.Generation <= 1 {
				newState = v1beta1.Deploying
			} else {
				newState = v1beta1.Updating
			}
		} else {
			newState = v1beta1.Down
		}
	default:
		newState = v1beta1.Running
		d.Status.DeployInProgress = false
		projectID := d.Labels[string(v1beta1.AppOrchActiveProjectID)]
		orchLibMetrics.RecordTimestamp(projectID, d.GetId(), d.Spec.DisplayName, string(newState), "status-change")
		if newState == v1beta1.Running {
			orchLibMetrics.CalculateTimeDifference(projectID, d.GetId(), d.Spec.DisplayName, "start", "CreateDeployment", string(v1beta1.Running), "status-change")
		}
	}

	d.Status.Display = fmt.Sprintf("Clusters: %v/%v/%v/%v, Apps: %v", clustercounts.Total, clustercounts.Running,
		clustercounts.Down, clustercounts.Unknown, apps)
	d.Status.Message = message
	d.Status.Summary = clustercounts
	d.Status.State = newState
}

func getGitRepoName(appName string, depID string) string {
	return fmt.Sprintf("%s-%s", appName, depID)
}

func getAppNameForGitRepo(gitrepo *fleetv1alpha1.GitRepo, depID string) string {
	suffix := fmt.Sprintf("-%s", depID)
	return strings.TrimSuffix(gitrepo.Name, suffix)
}
