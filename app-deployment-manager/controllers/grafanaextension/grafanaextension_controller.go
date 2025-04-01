// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package grafanaextension

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/atomix/atomix/api/errors"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/grafana"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	retryUpdateCounter = 10
	retryInterval      = 1 * time.Second
)

// Reconciler reconciles a GrafanaExtension object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=grafanaextensions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=grafanaextensions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=grafanaextensions/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("GrafanaExtensionController: grafana reconcile running")

	gext := &appv1beta1.GrafanaExtension{}
	if err := r.Client.Get(ctx, req.NamespacedName, gext); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found; return early
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	patch := client.MergeFrom(gext.DeepCopy())

	// if DeletionTimeStamp is not zero, return nil error
	if !gext.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	errs := make([]error, 0)

	result, err := r.reconcile(ctx, gext)
	if err != nil {
		errs = append(errs, err)
	}

	patchErr := r.updateStatusAndWait(ctx, req, gext, patch)
	if patchErr != nil {
		log.Error(patchErr, "GrafanaExtensionController: failed to status update")
		errs = append(errs, patchErr)
	}

	return result, kerrors.NewAggregate(errs)
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, err := ctrl.NewControllerManagedBy(mgr).
		For(&appv1beta1.GrafanaExtension{}).
		Owns(&v1.ConfigMap{}).
		Build(r)
	if err != nil {
		return fmt.Errorf("failed to setup grafana extension controller (%+v)", err)
	}

	return nil
}

func (r *Reconciler) reconcile(ctx context.Context, gext *appv1beta1.GrafanaExtension) (ctrl.Result, error) {
	phases := []func(ctx context.Context, gext *appv1beta1.GrafanaExtension) (ctrl.Result, error){
		r.reconcileGrafanaConfigmap,
		r.reconcileDashboardState,
	}

	for _, phase := range phases {
		_, err := phase(ctx, gext)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// reconcileGrafanaConfigmap is the 'reconcile' function to create the configmap to import dashboard JSON model to Grafana
func (r *Reconciler) reconcileGrafanaConfigmap(ctx context.Context, gext *appv1beta1.GrafanaExtension) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("GrafanaExtensionController: reconcile Grafana ConfigMap")

	if gext.Status.GetCondition(appv1beta1.GrafanaExtensionConditionConfigMapReady).Status == metav1.ConditionTrue &&
		(gext.Generation == 1 || gext.Generation == gext.Status.ReconciledGeneration) {
		log.Info("GrafanaExtensionController: skip reconcile Grafana ConfigMap because the configmap is already created before")
		return ctrl.Result{}, nil
	}

	// JSON file name for Grafana JSON model
	jsonFileName := fmt.Sprintf("%s-%s.json", gext.UID, gext.Name)
	// Define ConfigMap to add Grafana dashboard
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", gext.Name, gext.UID),
			Namespace:   gext.Namespace,
			Labels:      map[string]string{appv1beta1.GrafanaExtensionLabelDashboardKey: appv1beta1.GrafanaExtensionLabelDashboard, appv1beta1.GrafanaExtensionLabelProjectKey: gext.Spec.Project},
			Annotations: map[string]string{appv1beta1.GrafanaExtensionFolderAnnotationKey: appv1beta1.GrafanaExtensionDefaultFolder},
		},
		Data: map[string]string{jsonFileName: gext.Spec.ArtifactRef.Artifact},
	}

	if err := cutil.SetControllerReference(gext, cm, r.Scheme); err != nil {
		log.Error(err, "Failed to set Grafana configmap controller reference")
		gext.Status.SetStatus(time.Now(), appv1beta1.Down, fmt.Sprintf("Error: %s", err.Error()), appv1beta1.GrafanaExtensionConditionConfigMapReady, metav1.ConditionFalse, gext.Status.ReconciledGeneration)
		return ctrl.Result{}, err
	}

	if gext.Generation == 1 {
		// create case
		if err := r.Client.Create(ctx, cm); err != nil {
			log.Error(err, "GrafanaExtensionController: failed to create Grafana dashboard ConfigMap")
			// update state to add condition with error message
			gext.Status.SetStatus(time.Now(), appv1beta1.Down, fmt.Sprintf("Error: %s", err.Error()), appv1beta1.GrafanaExtensionConditionConfigMapReady, metav1.ConditionFalse, gext.Status.ReconciledGeneration)
			return ctrl.Result{}, err
		}
	} else if gext.Generation != gext.Status.ReconciledGeneration {
		// update case
		if err := r.Client.Update(ctx, cm); err != nil {
			log.Error(err, "GrafanaExtensionController: failed to update Grafana dashboard ConfigMap")
			// update state to add condition with error message
			gext.Status.SetStatus(time.Now(), appv1beta1.Down, fmt.Sprintf("Error: %s", err.Error()), appv1beta1.GrafanaExtensionConditionConfigMapReady, metav1.ConditionFalse, gext.Status.ReconciledGeneration)
			return ctrl.Result{}, err
		}
	}

	gext.Status.SetStatus(time.Now(), appv1beta1.Down, "Completed", appv1beta1.GrafanaExtensionConditionConfigMapReady, metav1.ConditionTrue, gext.Status.ReconciledGeneration)
	log.Info(fmt.Sprintf("dashboard configmap created: name - %s", fmt.Sprintf("%s-%s", gext.Name, gext.UID)))

	return ctrl.Result{}, nil
}

// reconcileDashboardState is the 'reconcile' function to check if the dashboard is imported successfully or not
func (r *Reconciler) reconcileDashboardState(ctx context.Context, gext *appv1beta1.GrafanaExtension) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("GrafanaExtensionController: reconcile Grafana dashboard state")

	if gext.Status.GetCondition(appv1beta1.GrafanaExtensionConditionDashboardReady).Status == metav1.ConditionTrue &&
		(gext.Generation == 1 || gext.Generation == gext.Status.ReconciledGeneration) {
		log.Info("GrafanaExtensionController: skip reconcile Grafana Dashboard because the dashboard is already verified")
		return ctrl.Result{}, nil
	}

	// start dashboard readiness check
	// get Grafana UID first
	funcGetGrafanaDashboardUIDMutex.Lock()
	uid, err := funcGetGrafanaDashboardUID(ctx, gext.Spec.ArtifactRef.Artifact)
	funcGetGrafanaDashboardUIDMutex.Unlock()
	if err != nil {
		// when failed to get UID from Grafana JSON model in artifact
		log.Error(err, "GrafanaExtensionController: failed to get UID from JSON model")
		gext.Status.SetStatus(time.Now(), appv1beta1.Down, fmt.Sprintf("Error: %s", err.Error()), appv1beta1.GrafanaExtensionConditionDashboardReady, metav1.ConditionFalse, gext.Status.ReconciledGeneration)

		return ctrl.Result{}, err
	}

	// Check Grafana dashboard is ready
	funcIsGrafanaDashboardReadyMutex.Lock()
	err = funcIsGrafanaDashboardReady(ctx, uid)
	funcIsGrafanaDashboardReadyMutex.Unlock()
	if err != nil {
		errMsg := fmt.Sprintf("dashboard UID %s not found; err: %s", uid, err.Error())
		gext.Status.SetStatus(time.Now(), appv1beta1.Down, fmt.Sprintf("Error: %s", errMsg), appv1beta1.GrafanaExtensionConditionDashboardReady, metav1.ConditionFalse, gext.Status.ReconciledGeneration)

		return ctrl.Result{}, errors.NewNotFound(errMsg)
	}

	// update status Running; once it is running, update reconcileGeneration
	gext.Status.SetStatus(time.Now(), appv1beta1.Running, "Complete", appv1beta1.GrafanaExtensionConditionDashboardReady, metav1.ConditionTrue, gext.GetGeneration())
	log.Info("Grafana dashboard verified")

	return ctrl.Result{}, nil
}

// updateStatusAndWait is the function to update Kubernetes object status and wait until it is really applied.
// after calling r.Client.Status.Update, the new value is updated after some time - there is some delay.
// Thus, add a logic to wait until the new value/status is really applied.
// If timeout timer is expired, raise a timeout error.
func (r *Reconciler) updateStatusAndWait(ctx context.Context, req ctrl.Request, newExt *appv1beta1.GrafanaExtension, patch client.Patch) error {
	patchErr := r.Client.Status().Patch(ctx, newExt, patch)
	if patchErr != nil {
		return patchErr
	}

	for i := 0; i < retryUpdateCounter; i++ {
		orig := &appv1beta1.GrafanaExtension{}
		if err := r.Client.Get(ctx, req.NamespacedName, orig); err != nil {
			return err
		}

		if reflect.DeepEqual(orig.Status, newExt.Status) {
			return nil
		}
		time.Sleep(retryInterval)
	}

	return errors.NewTimeout("timeout timer expired when waiting status updated")
}

var ( // for unit tests
	funcGetGrafanaDashboardUID       = grafana.GetGrafanaDashboardUID
	funcIsGrafanaDashboardReady      = grafana.IsGrafanaDashboardReady
	funcGetGrafanaDashboardUIDMutex  = sync.RWMutex{}
	funcIsGrafanaDashboardReadyMutex = sync.RWMutex{}
)
