// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deploymentcluster

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	ctrlmetrics "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/metrics"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	orchLibMetrics "github.com/open-edge-platform/orch-library/go/pkg/metrics"
)

var (
	clusterKey = "metadata.clusterNamespacedName"

	deploymentClusterPredicate = predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool {
			// process
			return true
		},
		DeleteFunc: func(_ event.DeleteEvent) bool {
			// no action
			return false
		},
		UpdateFunc: func(_ event.UpdateEvent) bool {
			// no action
			return false
		},
		GenericFunc: func(_ event.GenericEvent) bool {
			// no action
			return false
		},
	}

	clusterStatePredicate = predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool {
			// process
			return true
		},
		DeleteFunc: func(_ event.DeleteEvent) bool {
			// process
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// process if the original object has different state than the new one
			old := e.ObjectOld.(*v1beta1.Cluster).Status.State
			next := e.ObjectNew.(*v1beta1.Cluster).Status.State
			return (old != next) && (old == v1beta1.Unknown || next == v1beta1.Unknown)
		},
		GenericFunc: func(_ event.GenericEvent) bool {
			// no action
			return false
		},
	}
)

// Reconciler reconciles a DeploymentCluster object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type ClusterInfo struct {
	Name             string
	DeploymentID     string
	ClusterID        string
	ClusterNamespace string
}

//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=deploymentclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=deploymentclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=deploymentclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DeploymentCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	condstatus := v1.ConditionFalse
	reason := "BundleDeploymentsNotReady"
	log := log.FromContext(ctx)
	log.Info("Test Reconciling DeploymentCluster")
	dc := &v1beta1.DeploymentCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, dc); err != nil {
		if apierrors.IsNotFound(err) {
			// Reconcile loop will retrigger for new DeploymentCluster
			err = r.createDeploymentCluster(ctx, req)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	patch := client.MergeFrom(dc.DeepCopy())

	// Initialize the status of this DeploymentCluster because it will be rebuilt
	initializeStatus(dc)

	// Get Cluster info for this DeploymentCluster
	cluster := v1beta1.Cluster{}
	key := types.NamespacedName{
		Namespace: dc.Spec.Namespace,
		Name:      dc.Spec.ClusterID,
	}

	if err := r.Get(ctx, key, &cluster); err != nil {
		if apierrors.IsNotFound(err) {
			// Delete this DeploymentCluster since Cluster is not found
			err = r.deleteDeploymentCluster(ctx, dc)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	dc.Status.Name = cluster.Spec.DisplayName

	// Get all BundleDeployments in the cluster namespace for the appropriate Deployment
	bdlist := &fleetv1alpha1.BundleDeploymentList{}
	labels := map[string]string{
		string(v1beta1.DeploymentID): dc.Labels[string(v1beta1.DeploymentID)],
		string(v1beta1.BundleType):   fleet.BundleTypeApp.String(),
	}
	if err := r.List(ctx, bdlist, client.InNamespace(req.Namespace), client.MatchingLabels(labels)); err != nil {
		return ctrl.Result{}, err
	}

	// Walk the BundleDeployment list and update DeploymentCluster status
	for i := range bdlist.Items {
		bd := &bdlist.Items[i]
		dci, err := deploymentClusterInfo(bd)
		fmt.Println("Test deploymentClusterInfo: ", dci.Name, req.Name, err)
		if err == nil && dci.Name == req.Name {
			addDeploymentClusterApp(bd, dc)
		}
	}

	if dc.Status.Status.Summary.Total == 0 {
		// Delete this DeploymentCluster since it has no Apps
		err := r.deleteDeploymentCluster(ctx, dc)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if cluster.Status.State == v1beta1.Unknown {
		// Cluster is offline
		reason = "ClusterStatusUnknown"
		initializeStatus(dc)
		dc.Status.Name = cluster.Spec.DisplayName
		dc.Status.Status.State = v1beta1.Unknown
	}

	if dc.Status.Status.State == v1beta1.Running {
		condstatus = v1.ConditionTrue
		reason = "BundleDeploymentsReady"
	}
	dc.Status.Conditions = utils.UpdateStatusCondition(dc.Status.Conditions, "Ready", condstatus, reason, nil)

	// Update this DeploymentCluster's status
	dc.Status.Display = fmt.Sprintf("%d/%d", dc.Status.Status.Summary.Running, dc.Status.Status.Summary.Total)
	dc.Status.LastStatusUpdate = v1.Time{Time: time.Now().UTC()}

	if err := r.Status().Patch(ctx, dc, patch); err != nil {
		log.Error(err, "Failed to update DeploymentCluster")
		return ctrl.Result{}, err
	}

	updateStatusMetrics(ctx, r, dc, false)

	return ctrl.Result{}, nil
}

func updateStatusMetrics(ctx context.Context, r *Reconciler, dc *v1beta1.DeploymentCluster, deleteMetrics bool) {
	metricValue := make(map[string]float64)
	log := log.FromContext(ctx)

	// Init all values to 0
	metricValue[string(v1beta1.Running)] = 0
	metricValue[string(v1beta1.Down)] = 0
	metricValue[string(v1beta1.Unknown)] = 0

	projectID := ""
	if _, ok := dc.Labels[string(v1beta1.AppOrchActiveProjectID)]; ok {
		projectID = dc.Labels[string(v1beta1.AppOrchActiveProjectID)]
	}

	// Fetch the Deployment object using DeploymentID
	deploymentID := dc.Spec.DeploymentID
	displayName, err := getDisplayName(ctx, projectID, deploymentID, r)
	if err != nil {
		log.Error(err, "Couldnt get displayName for deployment")
	}

	if deleteMetrics {
		// Delete current deployment metrics only
		for i := range metricValue {
			ctrlmetrics.DeploymentClusterStatus.DeleteLabelValues(projectID, dc.Spec.DeploymentID, displayName, dc.Spec.ClusterID, dc.Status.Name, i)
		}
	} else {
		// Only one status will be 1 and rest are 0
		metricValue[string(dc.Status.Status.State)] = 1

		// Update and output all metrics
		for i, val := range metricValue {
			ctrlmetrics.DeploymentClusterStatus.WithLabelValues(projectID, dc.Spec.DeploymentID, displayName, dc.Spec.ClusterID, dc.Status.Name, i).Set(val)
		}

		dcStatusChange := "dc-" + dc.Spec.ClusterID + "-status-change"
		orchLibMetrics.RecordTimestamp(projectID, dc.Spec.DeploymentID, displayName, string(dc.Status.Status.State), dcStatusChange)
		if metricValue[string(v1beta1.Running)] == 1 {
			orchLibMetrics.CalculateTimeDifference(projectID, dc.Spec.DeploymentID, displayName, "start", "CreateDeployment", string(v1beta1.Running), dcStatusChange)
		}

	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add field indexer with metadata.clusterNamespacedName as field name and the associated cluster identifier as value.
	// Combine namespace and name into a single IndexField due to cache limitations.
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1beta1.DeploymentCluster{}, clusterKey,
		func(rawObj client.Object) []string {
			dc := rawObj.(*v1beta1.DeploymentCluster)
			return []string{getNamespacedName(dc.Spec.Namespace, dc.Spec.ClusterID)}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1beta1.DeploymentCluster{},
			builder.WithPredicates(deploymentClusterPredicate),
		).
		Watches(
			&fleetv1alpha1.BundleDeployment{},
			handler.EnqueueRequestsFromMapFunc(r.triggerReconcileBD),
		).
		Watches(
			&v1beta1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(r.triggerReconcileCluster),
			builder.WithPredicates(clusterStatePredicate),
		).
		Complete(r)
}

func (r *Reconciler) triggerReconcileBD(_ context.Context, o client.Object) []reconcile.Request {
	requests := []reconcile.Request{}
	bd, ok := o.(*fleetv1alpha1.BundleDeployment)
	if !ok {
		return requests
	}

	// Ignore init bundles
	if v, ok := bd.ObjectMeta.Labels[string(v1beta1.BundleType)]; ok && v == fleet.BundleTypeInit.String() {
		return requests
	}
	dci, err := deploymentClusterInfo(bd)
	if err == nil {
		requests = []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: bd.Namespace,
					Name:      dci.Name,
				},
			},
		}
	}
	return requests
}

func getNamespacedName(namespace string, name string) string {
	return namespace + "/" + name
}

func (r *Reconciler) triggerReconcileCluster(ctx context.Context, o client.Object) []reconcile.Request {
	log := log.FromContext(ctx)
	requests := []reconcile.Request{}
	cluster, ok := o.(*v1beta1.Cluster)
	if !ok {
		return requests
	}

	// Fetch the DeploymentClusters associated with this Cluster
	var dclist v1beta1.DeploymentClusterList
	key := getNamespacedName(cluster.Namespace, cluster.Name)
	if err := r.List(ctx, &dclist, client.MatchingFields{clusterKey: key}); err != nil {
		log.Error(err, "Failed to list DeploymentClusters")
		return requests
	}

	for _, dc := range dclist.Items {
		newreq := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: dc.Namespace,
				Name:      dc.Name,
			},
		}
		requests = append(requests, newreq)
	}

	return requests
}

func getDisplayName(ctx context.Context, projectID, deploymentID string, r *Reconciler) (string, error) {
	deploymentList := &v1beta1.DeploymentList{}
	log := log.FromContext(ctx)

	if err := r.Client.List(ctx, deploymentList, client.MatchingLabels{string(v1beta1.AppOrchActiveProjectID): projectID}); err != nil {
		log.Error(err, "Failed to list Deployments")
		return "", err
	}

	var deployment *v1beta1.Deployment
	for _, d := range deploymentList.Items {
		if d.ObjectMeta.UID == types.UID(deploymentID) {
			deployment = &d
			break
		}
	}

	if deployment == nil {
		log.Error(nil, "Deployment not found", "DeploymentID", deploymentID)
		return "", errors.New("Deployment not found")
	}

	displayName := deployment.Spec.DisplayName
	return displayName, nil
}

func (r *Reconciler) createDeploymentCluster(ctx context.Context, req ctrl.Request) error {
	log := log.FromContext(ctx)

	// Get all BundleDeployments in the DeploymentCluster namespace
	bdlist := &fleetv1alpha1.BundleDeploymentList{}
	if err := r.List(ctx, bdlist, client.InNamespace(req.Namespace)); err != nil {
		return err
	}

	// Check if any map to the DeploymentCluster in the request
	// If so, create a new DeploymentCluster for this BundleDeployment
	for i := range bdlist.Items {
		bd := &bdlist.Items[i]
		dci, err := deploymentClusterInfo(bd)
		if err == nil {
			if dci.Name == req.Name {
				dc := newDeploymentCluster(req, dci)

				if activeProjectID, ok := bd.Labels[string(v1beta1.AppOrchActiveProjectID)]; ok {
					dc.Labels[string(v1beta1.AppOrchActiveProjectID)] = activeProjectID
				} else {
					return fmt.Errorf("cannot create DeploymentCluster %s: active project ID label not found in BundleDeployment %s", dc.Name, bd.Name)
				}

				log.Info(fmt.Sprintf("Created DeploymentCluster %s", dc.Name))
				projectID := dc.Labels[string(v1beta1.AppOrchActiveProjectID)]

				displayName, err := getDisplayName(ctx, projectID, dc.Spec.DeploymentID, r)
				if err != nil {
					log.Error(err, "Couldnt get displayName for deployment")
				}
				dcForDeployment := "dc-" + dc.Spec.ClusterID
				orchLibMetrics.RecordTimestamp(projectID, dc.Spec.DeploymentID, displayName, "start", dcForDeployment)
				orchLibMetrics.CalculateTimeDifference(projectID, dc.Spec.DeploymentID, displayName, "start", "CreateDeployment", "start", dcForDeployment)

				return r.Client.Create(ctx, dc)
			}
		}
	}

	return nil
}

func (r *Reconciler) deleteDeploymentCluster(ctx context.Context, dc *v1beta1.DeploymentCluster) error {
	log := log.FromContext(ctx)

	if err := r.Client.Delete(ctx, dc); err != nil {
		log.Error(err, fmt.Sprintf("Failed to delete DeploymentCluster %s", dc.Name))
		return err
	}
	log.Info(fmt.Sprintf("Deleted DeploymentCluster %s", dc.Name))

	// Clean up terminated deployment status metrics
	updateStatusMetrics(ctx, r, dc, true)

	return nil
}

// deploymentClusterName returns the DeploymentCluster name associated with this BundleDeployment,
// along with the Deployment ID and Cluster ID read from the BundleDeployment labels.
//
// It constructs the DeploymentCluster name as "dc-<new UUID>", where the new UUID is generated from the
// Deployment and Cluster IDs. The "dc-" prefix is there because resource names need to start with a letter
// (i.e., DNS-1035 label, regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?'). The resulting name
// will always be 39 characters.}
func deploymentClusterInfo(bd *fleetv1alpha1.BundleDeployment) (ClusterInfo, error) {
	deplid, ok := bd.Labels[string(v1beta1.DeploymentID)]
	if !ok {
		return ClusterInfo{}, errors.New("deployment ID label not found in BundleDeployment")
	}
	clusterid, ok := bd.Labels[string(v1beta1.FleetClusterID)]
	if !ok {
		return ClusterInfo{}, errors.New("cluster ID label not found in BundleDeployment")
	}
	clusternamespace, ok := bd.Labels[string(v1beta1.FleetClusterNamespace)]
	if !ok {
		return ClusterInfo{}, errors.New("cluster namespace label not found in BundleDeployment")
	}

	u := uuid.MustParse(deplid)
	next := uuid.NewSHA1(u, []byte(clusterid))

	name := fmt.Sprintf("dc-%v", next)
	dci := ClusterInfo{
		Name:             name,
		DeploymentID:     deplid,
		ClusterID:        clusterid,
		ClusterNamespace: clusternamespace,
	}
	return dci, nil
}

func newDeploymentCluster(req ctrl.Request, dci ClusterInfo) *v1beta1.DeploymentCluster {
	dc := &v1beta1.DeploymentCluster{
		ObjectMeta: v1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Labels: map[string]string{
				string(v1beta1.DeploymentID): dci.DeploymentID,
				string(v1beta1.ClusterName):  dci.ClusterID,
			},
		},
		Spec: v1beta1.DeploymentClusterSpec{
			DeploymentID: dci.DeploymentID,
			ClusterID:    dci.ClusterID,
			Namespace:    dci.ClusterNamespace,
		},
	}
	return dc
}

func initializeStatus(dc *v1beta1.DeploymentCluster) {
	dc.Status = v1beta1.DeploymentClusterStatus{
		Status: v1beta1.Status{
			State:   v1beta1.Running,
			Message: "",
			Summary: v1beta1.Summary{
				Type: v1beta1.AppCounts,
			},
		},
		Apps:       []v1beta1.App{},
		Conditions: dc.Status.Conditions,
	}
}

// addDeploymentClusterApp updates the DeploymentCluster's list of Apps with info from this BundleDeployment
func addDeploymentClusterApp(bd *fleetv1alpha1.BundleDeployment, dc *v1beta1.DeploymentCluster) {
	var state v1beta1.StateType

	fmt.Println("Test addDeploymentClusterApp: ", utils.GetDeploymentGeneration(bd))
	switch utils.GetState(bd) {
	case v1beta1.Running:
		state = v1beta1.Running
	default:
		state = v1beta1.Down
	}

	app := v1beta1.App{
		Name:                 utils.GetAppName(bd),
		Id:                   utils.GetAppID(bd),
		DeploymentGeneration: utils.GetDeploymentGeneration(bd),
		Status: v1beta1.Status{
			State:   state,
			Message: utils.GetMessage(bd),
			Summary: v1beta1.Summary{
				Type: v1beta1.NotUsed,
			},
		},
	}
	dc.Status.Status.Summary.Total++
	if app.Status.State == v1beta1.Running {
		dc.Status.Status.Summary.Running++
	} else {
		dc.Status.Status.Summary.Down++
		dc.Status.Status.State = v1beta1.Down
		dc.Status.Status.Message = utils.AppendMessage(dc.Status.Status.Message, app.Status.Message)
	}
	dc.Status.Apps = append(dc.Status.Apps, app)
}
