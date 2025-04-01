// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller"
	networkv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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

const networkClusterStatusFinalizer = "networkclusterstatus.network.app.edge-orchestrator.intel.com/finalizer"

func AddNetworkClusterStatusController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &NetworkClusterStatusController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// Ignore stuttering type name due to multiple controllers being in the same package.
//revive:disable

// NetworkClusterStatusController is a Controller for setting up the Cluster
type NetworkClusterStatusController struct {
	*controller.Controller
}

//revive:enable

func (c *NetworkClusterStatusController) Setup(mgr manager.Manager) error {
	log.Info("Setting up networkclusterstatus-controller")
	controller, err := controllerruntime.New("networkclusterstatus-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&networkv1alpha1.NetworkService{},
		&handler.TypedEnqueueRequestForObject[*networkv1alpha1.NetworkService]{}))
	if err != nil {
		log.Error(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&networkv1alpha1.NetworkCluster{},
		handler.TypedEnqueueRequestsFromMapFunc[*networkv1alpha1.NetworkCluster, reconcile.Request](func(ctx context.Context, networkCluster *networkv1alpha1.NetworkCluster) []reconcile.Request {
			var networkServices networkv1alpha1.NetworkServiceList
			if err := c.List(ctx, &networkServices, client.MatchingLabels{
				networkNameKey:      networkCluster.Spec.NetworkRef.Name,
				clusterNamespaceKey: networkCluster.Spec.ClusterRef.Namespace,
				clusterNameKey:      networkCluster.Spec.ClusterRef.Name,
			}); err != nil {
				log.Error(err)
				return nil
			}

			var requests []reconcile.Request
			for _, networkService := range networkServices.Items {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: networkService.Name,
					},
				})
			}
			return requests
		})))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *NetworkClusterStatusController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var networkService networkv1alpha1.NetworkService
	if err := c.Get(ctx, request.NamespacedName, &networkService); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	log.Infof("Reconciling NetworkClusterStatus %s", networkService.Spec.ClusterRef.Name)

	var networkCluster networkv1alpha1.NetworkCluster
	networkClusterKey := client.ObjectKey{
		Name: newNetworkClusterName(networkService.Spec.NetworkRef, networkService.Spec.ClusterRef),
	}
	if err := c.Get(ctx, networkClusterKey, &networkCluster); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		log.Warnf("NetworkCluster %s not found", networkClusterKey.Name)

		if networkService.DeletionTimestamp != nil && controllerutil.RemoveFinalizer(&networkService, networkClusterStatusFinalizer) {
			if err := c.Update(ctx, &networkService); err != nil && !errors.IsNotFound(err) {
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

	if networkCluster.Status.Role == networkv1alpha1.NetworkClusterRoleUnknown {
		log.Infof("Network %s: Changing NetworkCluster %s role to %s",
			networkCluster.Spec.NetworkRef.Name, networkCluster.Name, networkv1alpha1.NetworkClusterRoleSpoke)
		networkCluster.Status.Role = networkv1alpha1.NetworkClusterRoleSpoke
		if err := c.Status().Update(ctx, &networkCluster); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return reconcile.Result{}, err
		}
	}

	if networkService.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(&networkService, networkClusterStatusFinalizer) {
			var newServices []corev1.LocalObjectReference
			for _, service := range networkCluster.Status.Services {
				if service.Name != networkService.Name {
					newServices = append(newServices, service)
				}
			}

			if len(newServices) != len(networkCluster.Status.Services) {
				log.Infof("Removing NetworkService %s from NetworkClusterStatus %s", networkService.Name, networkCluster.Name)
				networkCluster.Status.Services = newServices
				if len(newServices) == 0 {
					log.Infof("Changing NetworkCluster %s Role to %s", networkService.Spec.ClusterRef.Name, networkv1alpha1.NetworkClusterRoleSpoke)
					networkCluster.Status.Role = networkv1alpha1.NetworkClusterRoleSpoke
				}
				if err := c.Status().Update(ctx, &networkCluster); err != nil {
					if !errors.IsNotFound(err) {
						log.Error(err)
						return reconcile.Result{}, err
					}
				}
			}

			log.Infof("Removing finalizer %s from NetworkService %s", networkClusterStatusFinalizer, networkService.Name)
			if err := c.Update(ctx, &networkService); err != nil && !errors.IsNotFound(err) {
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

	if controllerutil.AddFinalizer(&networkService, networkClusterStatusFinalizer) {
		log.Infof("Adding finalizer %s to Service %s", networkClusterStatusFinalizer, networkService.Name)
		if err := c.Update(ctx, &networkService); err != nil {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Check if the service is already logged in the cluster status.
	for _, networkServiceRef := range networkCluster.Status.Services {
		if networkServiceRef.Name == networkService.Name {
			return reconcile.Result{}, nil
		}
	}

	// If the service is not found in the cluster status, add it.
	log.Infof("Adding NetworkService %s to NetworkClusterStatus %s", networkService.Name, networkCluster.Name)
	networkCluster.Status.Services = append(networkCluster.Status.Services, corev1.LocalObjectReference{
		Name: networkService.Name,
	})
	if networkCluster.Status.Role != networkv1alpha1.NetworkClusterRoleHub {
		log.Infof("Changing NetworkCluster %s Role to %s", networkService.Spec.ClusterRef.Name, networkv1alpha1.NetworkClusterRoleHub)
		networkCluster.Status.Role = networkv1alpha1.NetworkClusterRoleHub
	}
	if err := c.Status().Update(ctx, &networkCluster); err != nil && !errors.IsNotFound(err) {
		if errors.IsConflict(err) {
			log.Warn(err)
		} else {
			log.Error(err)
		}
	}
	return reconcile.Result{}, nil
}
