// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interconnect

import (
	"context"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/utils"
	skupperlib "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper"
	skupperutils "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/utils/skupper"
	interconnectv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/interconnect/v1alpha1"
	errorutils "github.com/open-edge-platform/orch-library/go/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/workqueue"
	controllerruntime "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

const clusterFinalizer = "cluster.interconnect.app.edge-orchestrator.intel.com/finalizer"

func AddClusterController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &ClusterController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// ClusterController is a Controller for setting up the Cluster
type ClusterController struct {
	*controller.Controller
}

func (c *ClusterController) Setup(mgr manager.Manager) error {
	log.Info("Setting up cluster-controller")
	controller, err := controllerruntime.New("cluster-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&interconnectv1alpha1.Cluster{},
		&handler.TypedEnqueueRequestForObject[*interconnectv1alpha1.Cluster]{}))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *ClusterController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling Cluster %s", request.NamespacedName)
	var cluster interconnectv1alpha1.Cluster
	if err := c.Get(ctx, request.NamespacedName, &cluster); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	projectID := clusterclient.ProjectID(cluster.Labels[utils.InterconnectProjectIDLabel])
	clusterID := clusterclient.ClusterID(cluster.Name)

	if cluster.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(&cluster, clusterFinalizer) {
			if cluster.Status.Phase != interconnectv1alpha1.ClusterTerminating {
				cluster.Status.Phase = interconnectv1alpha1.ClusterTerminating
				if err := c.Status().Update(ctx, &cluster); err != nil {
					if !errors.IsNotFound(err) {
						log.Error(err)
						return reconcile.Result{}, err
					}
				}
				return reconcile.Result{}, nil
			}
		}
	} else if !controllerutil.ContainsFinalizer(&cluster, clusterFinalizer) {
		controllerutil.AddFinalizer(&cluster, clusterFinalizer)
		if err := c.Update(ctx, &cluster); err != nil {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	switch cluster.Status.Phase {
	case interconnectv1alpha1.ClusterPending:
		// Move to the Configuring phase
		cluster.Status.Phase = interconnectv1alpha1.ClusterConfiguring
		if err := c.Status().Update(ctx, &cluster); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
		}
	case interconnectv1alpha1.ClusterConfiguring:
		clusterConfig, err := c.Clusters.GetClusterConfig(ctx, clusterID, projectID)
		if err != nil {
			log.Warn(err)
			return reconcile.Result{}, err
		}
		err = skupperlib.SkupperDelete(ctx, clusterConfig)
		if err == nil {
			log.Info("Deleting Skupper to Reconfigure, ", "clusterID: ", clusterID)
			return reconcile.Result{
				RequeueAfter: time.Second * 10,
			}, err
		}

		if err != nil && strings.Contains(err.Error(), "Skupper not installed") {
			ingressType := skupperutils.IngressNone
			if cluster.Status.Ingress == interconnectv1alpha1.IngressLoadBalancer {
				ingressType = skupperutils.IngressLoadBalancer
			}

			err = skupperlib.SkupperInit(ctx, clusterConfig, ingressType)
			if err != nil {
				log.Warn(err)
				return reconcile.Result{}, err
			}

			log.Info("Skupper initialized, ", "clusterID: ", clusterID)
			cluster.Status.Phase = interconnectv1alpha1.ClusterRunning
			if err := c.Status().Update(ctx, &cluster); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err)
					return reconcile.Result{}, err
				}
			}
		}
	case interconnectv1alpha1.ClusterRunning:
		// If necessary we can move the link and expose logic to the cluster's running phase.
	case interconnectv1alpha1.ClusterTerminating:
		clusterConfig, err := c.Clusters.GetClusterConfig(ctx, clusterID, projectID)
		if err != nil {
			log.Warn(err)
			if errorutils.IsNotFound(errorutils.FromGRPC(err)) {
				// Cluster is already deleted, remove finalizer
				controllerutil.RemoveFinalizer(&cluster, clusterFinalizer)
				if err := c.Update(ctx, &cluster); err != nil {
					if !errors.IsNotFound(err) && !errors.IsConflict(err) {
						log.Error(err)
						return reconcile.Result{}, err
					}
					return reconcile.Result{}, nil
				}
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, err
		}

		err = skupperlib.SkupperDelete(ctx, clusterConfig)
		if err != nil && !strings.Contains(err.Error(), "Skupper not installed") {
			log.Warn(err)
			return reconcile.Result{}, err
		}

		controllerutil.RemoveFinalizer(&cluster, clusterFinalizer)
		if err := c.Update(ctx, &cluster); err != nil {
			if !errors.IsNotFound(err) && !errors.IsConflict(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}
	}
	return reconcile.Result{}, nil
}
