// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/utils"
	networkv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const networkClusterLinkFinalizer = "networkclusterlink.network.app.edge-orchestrator.intel.com/finalizer"

func AddNetworkClusterLinkController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &NetworkClusterLinkController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// Ignore stuttering type name due to multiple controllers being in the same package.
//revive:disable

// NetworkClusterLinkController is a Controller for setting up the NetworkCluster
type NetworkClusterLinkController struct {
	*controller.Controller
}

//revive:enable

func (c *NetworkClusterLinkController) Setup(mgr manager.Manager) error {
	log.Info("Setting up networkclusterlink-controller")
	controller, err := controllerruntime.New("networkclusterlink-controller", mgr, controllerruntime.Options{
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
		&networkv1alpha1.NetworkCluster{},
		handler.TypedEnqueueRequestsFromMapFunc[*networkv1alpha1.NetworkCluster, reconcile.Request](func(ctx context.Context, networkCluster *networkv1alpha1.NetworkCluster) []reconcile.Request {
			var networkClusterList networkv1alpha1.NetworkClusterList
			if err := c.List(ctx, &networkClusterList, client.MatchingLabels{
				networkNameKey: networkCluster.Spec.NetworkRef.Name,
			}); err != nil {
				log.Error(err)
				return nil
			}

			var requests []reconcile.Request
			for _, networkHubCluster := range networkClusterList.Items {
				if networkHubCluster.Name != networkCluster.Name && networkHubCluster.Status.Role == networkv1alpha1.NetworkClusterRoleHub {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      networkHubCluster.Name,
							Namespace: networkHubCluster.Namespace,
						},
					})
				}
			}
			return requests
		})))
	if err != nil {
		log.Error(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&networkv1alpha1.NetworkLink{},
		handler.TypedEnqueueRequestForOwner[*networkv1alpha1.NetworkLink](
			c.Scheme, c.RESTMapper(), &networkv1alpha1.NetworkCluster{})))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *NetworkClusterLinkController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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
		if controllerutil.RemoveFinalizer(&networkCluster, networkClusterLinkFinalizer) {
			if err := c.removeLinks(ctx, &networkCluster); err != nil {
				return reconcile.Result{}, err
			}

			log.Infof("Removing finalizer %s from NetworkCluster %s", networkClusterLinkFinalizer, networkCluster.Name)
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

	if controllerutil.AddFinalizer(&networkCluster, networkClusterLinkFinalizer) {
		log.Infof("Adding finalizer %s to NetworkCluster %s", networkClusterLinkFinalizer, networkCluster.Name)
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

	switch networkCluster.Status.Role {
	case networkv1alpha1.NetworkClusterRoleSpoke:
		if err := c.removeLinks(ctx, &networkCluster); err != nil {
			return reconcile.Result{}, err
		}
	case networkv1alpha1.NetworkClusterRoleHub:
		if err := c.addLinks(ctx, &networkCluster); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (c *NetworkClusterLinkController) addLinks(ctx context.Context, networkLinkHub *networkv1alpha1.NetworkCluster) error {
	var networkClusterList networkv1alpha1.NetworkClusterList
	if err := c.List(ctx, &networkClusterList, client.MatchingLabels{
		networkNameKey: networkLinkHub.Spec.NetworkRef.Name,
	}); err != nil {
		log.Error(err)
		return err
	}

	networkLinkRefs := make(map[corev1.LocalObjectReference]bool)
	for _, networkLinkRef := range networkLinkHub.Status.Links {
		networkLinkRefs[networkLinkRef] = true
	}

	activeLinkRefs := make(map[corev1.LocalObjectReference]bool)
	for _, networkCluster := range networkClusterList.Items {
		if networkCluster.Name == networkLinkHub.Name {
			continue
		}

		networkLinkRef := corev1.LocalObjectReference{
			Name: newNetworkLinkName(
				networkLinkHub.Spec.NetworkRef,
				networkLinkHub.Spec.ClusterRef,
				networkCluster.Spec.ClusterRef),
		}

		if !networkLinkRefs[networkLinkRef] {
			log.Infof("Updating NetworkCluster %s status", networkLinkHub.Name)
			networkLinkHub.Status.Links = append(networkLinkHub.Status.Links, networkLinkRef)
			if err := c.Status().Update(ctx, networkLinkHub); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return err
			}
		}

		if err := c.addLink(ctx, networkLinkHub, &networkCluster); err != nil {
			return err
		}
		activeLinkRefs[networkLinkRef] = true
	}

	var newLinkRefs []corev1.LocalObjectReference
	for _, networkLinkRef := range networkLinkHub.Status.Links {
		if !activeLinkRefs[networkLinkRef] {
			var networkLink networkv1alpha1.NetworkLink
			networkLinkKey := client.ObjectKey{
				Name: networkLinkRef.Name,
			}
			if err := c.Get(ctx, networkLinkKey, &networkLink); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err)
					return err
				}
			}
			if err := c.removeLink(ctx, networkLinkHub, networkLinkRef); err != nil {
				return err
			}
		} else {
			newLinkRefs = append(newLinkRefs, networkLinkRef)
		}
	}

	if len(newLinkRefs) != len(networkLinkHub.Status.Links) {
		log.Infof("Updating NetworkCluster %s status", networkLinkHub.Name)
		networkLinkHub.Status.Links = newLinkRefs
		if err := c.Status().Update(ctx, networkLinkHub); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return err
		}
	}
	return nil
}

func (c *NetworkClusterLinkController) addLink(ctx context.Context, networkLinkHub *networkv1alpha1.NetworkCluster, networkLinkSpoke *networkv1alpha1.NetworkCluster) error {
	var networkLink networkv1alpha1.NetworkLink
	networkLinkKey := client.ObjectKey{
		Name: newNetworkLinkName(
			networkLinkHub.Spec.NetworkRef,
			networkLinkHub.Spec.ClusterRef,
			networkLinkSpoke.Spec.ClusterRef),
	}
	if err := c.Get(ctx, networkLinkKey, &networkLink); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}

		log.Infof("Network %s: Creating network link %s [%s -> %s]",
			networkLinkHub.Spec.NetworkRef.Name, networkLinkKey.Name, networkLinkSpoke.Spec.ClusterRef.Name, networkLinkHub.Spec.ClusterRef.Name)
		networkLink = networkv1alpha1.NetworkLink{
			ObjectMeta: metav1.ObjectMeta{
				Name: networkLinkKey.Name,
				Labels: map[string]string{
					networkNameKey:              networkLinkHub.Spec.NetworkRef.Name,
					utils.NetworkProjectIDLabel: networkLinkHub.Labels[utils.NetworkProjectIDLabel],
				},
			},
			Spec: networkv1alpha1.NetworkLinkSpec{
				NetworkRef:       networkLinkHub.Spec.NetworkRef,
				SourceClusterRef: networkLinkSpoke.Spec.ClusterRef,
				TargetClusterRef: networkLinkHub.Spec.ClusterRef,
			},
		}

		if err := controllerutil.SetOwnerReference(networkLinkHub, &networkLink, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if err := c.Create(ctx, &networkLink); err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err)
			return err
		}
		return nil
	}
	return nil
}

func (c *NetworkClusterLinkController) removeLinks(ctx context.Context, networkLinkHub *networkv1alpha1.NetworkCluster) error {
	if len(networkLinkHub.Status.Links) == 0 {
		return nil
	}

	for _, networkLinkRef := range networkLinkHub.Status.Links {
		if err := c.removeLink(ctx, networkLinkHub, networkLinkRef); err != nil {
			return err
		}
	}

	log.Infof("Updating NetworkCluster %s status", networkLinkHub.Name)
	networkLinkHub.Status.Links = nil
	if err := c.Status().Update(ctx, networkLinkHub); err != nil && !errors.IsNotFound(err) {
		if errors.IsConflict(err) {
			log.Warn(err)
		} else {
			log.Error(err)
		}
		return err
	}
	return nil
}

func (c *NetworkClusterLinkController) removeLink(ctx context.Context, networkLinkHub *networkv1alpha1.NetworkCluster, networkLinkRef corev1.LocalObjectReference) error {
	var networkLink networkv1alpha1.NetworkLink
	networkLinkKey := client.ObjectKey{
		Name: networkLinkRef.Name,
	}
	if err := c.Get(ctx, networkLinkKey, &networkLink); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}
		return nil
	}

	log.Infof("Network %s: Deleting network link %s [%s -> %s]",
		networkLinkHub.Spec.NetworkRef.Name, networkLinkRef.Name, networkLink.Spec.SourceClusterRef.Name, networkLinkHub.Spec.ClusterRef.Name)
	if err := c.Delete(ctx, &networkLink); err != nil && !errors.IsNotFound(err) {
		if errors.IsConflict(err) {
			log.Warn(err)
		} else {
			log.Error(err)
		}
		return err
	}
	return nil
}

func newNetworkLinkName(networkRef, hubRef, spokeRef corev1.ObjectReference) string {
	networkLinkHash, err := hash(hubRef.Name, spokeRef.Name)
	if err != nil {
		log.Fatal(err)
		return ""
	}
	return fmt.Sprintf("%s-%s", networkRef.Name, networkLinkHash)
}
