// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"maps"
	"time"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/patch"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultFleetCheckinTimeMinute = 15
)

var (
	clusterPredicate = predicate.Funcs{
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
)

// Reconciler reconciles a Cluster object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme

	fleetCheckinTime time.Duration
	recorder         record.EventRecorder
}

//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=clusters/status,verbs=get;create;update;patch
//+kubebuilder:rbac:groups=app.edge-orchestrator.intel.com,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	// log.Info(fmt.Sprintf("ClusterController: cluster reconcile running: req - %+v", req))

	c := &v1beta1.Cluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, c); err != nil {
		if apierrors.IsNotFound(err) {

			// Don't create Cluster if Fleet Cluster was deleted
			fc := &fleetv1alpha1.Cluster{}
			if err := r.Client.Get(ctx, req.NamespacedName, fc); err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}

			// reconcile loop will re-trigger for new Cluster
			c = &v1beta1.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
			}
			if err := r.Client.Create(ctx, c); err != nil {
				return ctrl.Result{}, errors.NewUnavailable(fmt.Sprintf("Failed to create Cluster %+v not found, err: %+v", req, err))
			}
			return ctrl.Result{}, nil
		}
	}

	// Create patchHelper and record the current state of the deployment object
	patchHelper, err := patch.NewPatchHelper(c, r.Client)
	if err != nil {
		log.Error(err, "error creating patch helper")
		return ctrl.Result{}, err
	}

	// if DeletionTimeStamp is not zero, return nil error
	if !c.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	fc := &fleetv1alpha1.Cluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, fc); err != nil {
		if apierrors.IsNotFound(err) {
			// Fleet cluster is deleted or does not exist
			err = r.Client.Delete(ctx, c)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		log.Error(err, "ClusterController: failed to get fleet cluster")
		return ctrl.Result{}, err
	}

	// to make Cluster CR is deleted with Fleet Cluster, add owner reference
	if err := cutil.SetControllerReference(fc, c, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference between Fleet cluster and Cluster CR")
		return ctrl.Result{}, err
	}

	// create or update event - start sync process
	lastSeen := c.Status.FleetStatus.FleetAgentStatus.LastSeen
	r.synchronizeWithFleetCluster(c, fc)

	state := v1beta1.Running
	if !r.isClusterConnected(ctx, fc) {
		state = v1beta1.Unknown
	}
	c.Status.SetStatus(time.Now(), state, "Complete", v1beta1.ClusterConditionKubeconfig, v1.ConditionTrue, c.Generation, fc.Status, fc.Generation)

	err = patchHelper.Patch(ctx, c)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if LastSeen was updated; if so, schedule requeue after next expected update
	if lastSeen.Before(&c.Status.FleetStatus.FleetAgentStatus.LastSeen) {
		return ctrl.Result{RequeueAfter: r.fleetCheckinTime}, nil
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) isClusterConnected(ctx context.Context, fc *fleetv1alpha1.Cluster) bool {
	log := log.FromContext(ctx)
	if time.Since(fc.Status.Agent.LastSeen.Time) > r.fleetCheckinTime {
		log.Info(fmt.Sprintf("Cluster %s/%s is disconnected: lastSeen-%+v, currentTime-%+v, elapsedTime-%+v, allowed interval-%+v",
			fc.Name, fc.Namespace, fc.Status.Agent.LastSeen.Time, time.Now(), time.Since(fc.Status.Agent.LastSeen.Time), r.fleetCheckinTime))
		return false
	}

	return true
}

func (r *Reconciler) synchronizeWithFleetCluster(c *v1beta1.Cluster, fc *fleetv1alpha1.Cluster) {
	labels := make(map[string]string)
	maps.Copy(labels, fc.GetLabels())
	c.ObjectMeta.Labels = labels

	if activeProjectID, ok := c.ObjectMeta.Labels[v1beta1.ClusterOrchKeyProjectID]; ok {
		c.ObjectMeta.Labels[string(v1beta1.AppOrchActiveProjectID)] = activeProjectID
	}

	c.Spec.Name = fc.Name
	c.Spec.DisplayName = fc.Labels[string(v1beta1.ClusterName)]
	c.Spec.KubeConfigSecretName = fc.Spec.KubeConfigSecret
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {

	// setup fleet checkin interval time
	agentCheckinMinute, msg := utils.GetIntegerFromEnv("FLEET_AGENT_CHECKIN", defaultFleetCheckinTimeMinute)
	if msg != "" {
		ctrl.Log.Info(msg)
	}
	r.fleetCheckinTime = (time.Duration(agentCheckinMinute) * time.Minute)
	ctrl.Log.Info(fmt.Sprintf("Setting r.fleetCheckinTime to %+v", r.fleetCheckinTime))

	r.recorder = mgr.GetEventRecorderFor("controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1beta1.Cluster{},
			builder.WithPredicates(clusterPredicate),
		).
		Watches(&fleetv1alpha1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(r.triggerReconcile)).
		Complete(r)
}

func (r *Reconciler) triggerReconcile(_ context.Context, o client.Object) []reconcile.Request {
	fc := o.(*fleetv1alpha1.Cluster)

	requests := []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: fc.Namespace,
				Name:      fc.Name,
			},
		},
	}

	return requests
}
