// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interconnect

import (
	"context"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller"
	interconnectv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/interconnect/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerruntime "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

const clusterStatusFinalizer = "clusterstatus.interconnect.app.edge-orchestrator.intel.com/finalizer"

func AddClusterStatusController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &ClusterStatusController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// ClusterStatusController is a Controller for setting up the Cluster
type ClusterStatusController struct {
	*controller.Controller
}

func (c *ClusterStatusController) Setup(mgr manager.Manager) error {
	log.Info("Setting up clusterstatus-controller")
	controller, err := controllerruntime.New("clusterstatus-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&interconnectv1alpha1.Service{},
		&handler.TypedEnqueueRequestForObject[*interconnectv1alpha1.Service]{}))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *ClusterStatusController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var service interconnectv1alpha1.Service
	if err := c.Get(ctx, request.NamespacedName, &service); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	log.Infof("Reconciling ClusterStatus %s", service.Spec.ClusterRef.Name)

	var cluster interconnectv1alpha1.Cluster
	clusterKey := client.ObjectKey{
		Name: service.Spec.ClusterRef.Name,
	}
	if err := c.Get(ctx, clusterKey, &cluster); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		log.Warnf("Cluster %s not found", service.Spec.ClusterRef.Name)

		if service.DeletionTimestamp != nil && controllerutil.RemoveFinalizer(&service, clusterStatusFinalizer) {
			if err := c.Update(ctx, &service); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if service.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(&service, clusterStatusFinalizer) {
			var newServices []corev1.LocalObjectReference
			for _, serviceRef := range cluster.Status.Services {
				if serviceRef.Name != service.Name {
					newServices = append(newServices, serviceRef)
				}
			}

			if len(newServices) != len(cluster.Status.Services) {
				log.Infof("Removing Service %s from ClusterStatus %s", service.Name, cluster.Name)
				cluster.Status.Services = newServices
				if len(newServices) == 0 {
					log.Infof("Setting Cluster %s Ingress type to %s", service.Spec.ClusterRef.Name, interconnectv1alpha1.IngressNone)
					cluster.Status.Ingress = interconnectv1alpha1.IngressNone
				}
				if err := c.Status().Update(ctx, &cluster); err != nil {
					if !errors.IsNotFound(err) {
						log.Error(err)
						return reconcile.Result{}, err
					}
				}
			}

			log.Infof("Removing finalizer %s from Service %s", clusterStatusFinalizer, service.Name)
			if err := c.Update(ctx, &service); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if controllerutil.AddFinalizer(&service, clusterStatusFinalizer) {
		log.Infof("Adding finalizer %s to Service %s", clusterStatusFinalizer, service.Name)
		if err := c.Update(ctx, &service); err != nil {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Check if the service is already logged in the cluster status.
	found := false
	for _, serviceRef := range cluster.Status.Services {
		if serviceRef.Name == service.Name {
			found = true
			break
		}
	}

	// If the service is not found in the cluster status, add it.
	if !found {
		log.Infof("Adding Service %s to ClusterStatus %s", service.Name, cluster.Name)
		cluster.Status.Services = append(cluster.Status.Services, corev1.LocalObjectReference{
			Name: service.Name,
		})
		if cluster.Status.Ingress != interconnectv1alpha1.IngressLoadBalancer {
			log.Infof("Setting Cluster %s Ingress type to %s", service.Spec.ClusterRef.Name, interconnectv1alpha1.IngressLoadBalancer)
			cluster.Status.Ingress = interconnectv1alpha1.IngressLoadBalancer
			cluster.Status.Phase = interconnectv1alpha1.ClusterConfiguring
		}
		if err := c.Status().Update(ctx, &cluster); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
			log.Warnf("Cluster %s not found", service.Spec.ClusterRef.Name)
			return reconcile.Result{}, nil
		}
	}
	return reconcile.Result{}, nil
}
