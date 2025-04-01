// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/utils"
	interconnectv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/interconnect/v1alpha1"
	networkv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const networkClusterFinalizer = "networkcluster.network.app.edge-orchestrator.intel.com/finalizer"

func AddNetworkClusterController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &NetworkClusterController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// Ignore stuttering type name due to multiple controllers being in the same package.
//revive:disable

// NetworkClusterController is a Controller for setting up the NetworkCluster
type NetworkClusterController struct {
	*controller.Controller
}

//revive:enable

func (c *NetworkClusterController) Setup(mgr manager.Manager) error {
	log.Info("Setting up networkcluster-controller")
	controller, err := controllerruntime.New("networkcluster-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&networkv1alpha1.NetworkCluster{},
		&handler.TypedEnqueueRequestForObject[*networkv1alpha1.NetworkCluster]{}))
	if err != nil {
		log.Error(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&interconnectv1alpha1.Cluster{},
		handler.TypedEnqueueRequestForOwner[*interconnectv1alpha1.Cluster](
			c.Scheme, c.RESTMapper(), &networkv1alpha1.NetworkCluster{})))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *NetworkClusterController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling NetworkCluster %s", request.NamespacedName)
	var networkCluster networkv1alpha1.NetworkCluster
	if err := c.Get(ctx, request.NamespacedName, &networkCluster); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if networkCluster.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(&networkCluster, networkClusterFinalizer) {
			if err := c.unbindCluster(ctx, &networkCluster); err != nil {
				return reconcile.Result{}, err
			}

			log.Infof("Removing finalizer %s from NetworkCluster %s", networkClusterFinalizer, networkCluster.Name)
			if err := c.Update(ctx, &networkCluster); err != nil && !errors.IsNotFound(err) {
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

	if controllerutil.AddFinalizer(&networkCluster, networkClusterFinalizer) {
		log.Infof("Adding finalizer %s to NetworkCluster %s", networkClusterFinalizer, networkCluster.Name)
		if err := c.Update(ctx, &networkCluster); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if err := c.bindCluster(ctx, &networkCluster); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (c *NetworkClusterController) bindCluster(ctx context.Context, networkCluster *networkv1alpha1.NetworkCluster) error {
	var cluster interconnectv1alpha1.Cluster
	clusterKey := client.ObjectKey{Name: networkCluster.Spec.ClusterRef.Name}
	if err := c.Get(ctx, clusterKey, &cluster); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}

		log.Infof("Network %s: Creating interconnect Cluster %s",
			networkCluster.Spec.NetworkRef.Name, clusterKey.Name)
		cluster = interconnectv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterKey.Name,
				Labels: map[string]string{
					clusterNamespaceKey:              networkCluster.Spec.ClusterRef.Namespace,
					clusterNameKey:                   networkCluster.Spec.ClusterRef.Name,
					utils.InterconnectProjectIDLabel: networkCluster.Labels[utils.NetworkProjectIDLabel],
				},
			},
			Spec: interconnectv1alpha1.ClusterSpec{
				ClusterRef: networkCluster.Spec.ClusterRef,
			},
		}

		log.Infof("Network %s: Binding NetworkCluster %s to interconnect Cluster %s",
			networkCluster.Spec.NetworkRef.Name, networkCluster.Name, clusterKey.Name)
		if err := controllerutil.SetOwnerReference(networkCluster, &cluster, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if err := c.Create(ctx, &cluster); err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err)
			return err
		}
		return nil
	}

	if !hasOwnerRef(&cluster, networkCluster) {
		log.Infof("Network %s: Binding NetworkCluster %s to interconnect Cluster %s",
			networkCluster.Spec.NetworkRef.Name, networkCluster.Name, clusterKey.Name)
		if err := controllerutil.SetOwnerReference(networkCluster, &cluster, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if err := c.Update(ctx, &cluster); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return err
		}
		return nil
	}
	return nil
}

func (c *NetworkClusterController) unbindCluster(ctx context.Context, networkCluster *networkv1alpha1.NetworkCluster) error {
	var cluster interconnectv1alpha1.Cluster
	clusterKey := client.ObjectKey{Name: networkCluster.Spec.ClusterRef.Name}
	if err := c.Get(ctx, clusterKey, &cluster); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}
		return nil
	}

	if hasOwnerRef(&cluster, networkCluster) {
		log.Infof("Network %s: Unbinding NetworkCluster %s from interconnect Cluster %s",
			networkCluster.Spec.NetworkRef.Name, networkCluster.Name, clusterKey.Name)
		if err := controllerutil.RemoveOwnerReference(networkCluster, &cluster, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if len(cluster.OwnerReferences) == 0 {
			log.Infof("Network %s: Deleting interconnect Cluster %s",
				networkCluster.Spec.NetworkRef.Name, clusterKey.Name)
			if err := c.Delete(ctx, &cluster); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return err
			}
		} else {
			log.Infof("Network %s: Updating interconnect Cluster %s",
				networkCluster.Spec.NetworkRef.Name, clusterKey.Name)
			if err := c.Update(ctx, &cluster); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return err
			}
		}
	}
	return nil
}
