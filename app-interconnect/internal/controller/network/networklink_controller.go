// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/utils"
	interconnectv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/interconnect/v1alpha1"
	networkv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
	"sort"
	"time"
)

const networkLinkFinalizer = "networklink.network.app.edge-orchestrator.intel.com/finalizer"

func AddNetworkLinkController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &NetworkLinkController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// Ignore stuttering type name due to multiple controllers being in the same package.
//revive:disable

// NetworkLinkController is a Controller for setting up the NetworkLink
type NetworkLinkController struct {
	*controller.Controller
}

//revive:enable

func (c *NetworkLinkController) Setup(mgr manager.Manager) error {
	log.Info("Setting up networklink-controller")
	controller, err := controllerruntime.New("networklink-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&networkv1alpha1.NetworkLink{},
		&handler.TypedEnqueueRequestForObject[*networkv1alpha1.NetworkLink]{}))
	if err != nil {
		log.Error(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&interconnectv1alpha1.Link{},
		handler.TypedEnqueueRequestForOwner[*interconnectv1alpha1.Link](
			c.Scheme, c.RESTMapper(), &networkv1alpha1.NetworkLink{})))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *NetworkLinkController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling NetworkLink %s", request.NamespacedName)
	var networkLink networkv1alpha1.NetworkLink
	if err := c.Get(ctx, request.NamespacedName, &networkLink); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if networkLink.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(&networkLink, networkLinkFinalizer) {
			if err := c.unbindLink(ctx, &networkLink); err != nil {
				return reconcile.Result{}, err
			}

			log.Infof("Removing finalizer %s from NetworkLink %s", networkLinkFinalizer, networkLink.Name)
			if err := c.Update(ctx, &networkLink); err != nil && !errors.IsNotFound(err) {
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

	if controllerutil.AddFinalizer(&networkLink, networkLinkFinalizer) {
		log.Infof("Adding finalizer %s to NetworkLink %s", networkLinkFinalizer, networkLink.Name)
		if err := c.Update(ctx, &networkLink); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if err := c.bindLink(ctx, &networkLink); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (c *NetworkLinkController) bindLink(ctx context.Context, networkLink *networkv1alpha1.NetworkLink) error {
	var link interconnectv1alpha1.Link
	sourceName, targetName, linkName := newInterconnectLinkName(
		networkLink.Spec.SourceClusterRef,
		networkLink.Spec.TargetClusterRef)
	linkKey := client.ObjectKey{
		Name: linkName,
	}
	if err := c.Get(ctx, linkKey, &link); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}

		log.Infof("Network %s: Creating interconnect link %s [%s -> %s]",
			networkLink.Spec.NetworkRef.Name, linkName, networkLink.Spec.SourceClusterRef.Name, networkLink.Spec.TargetClusterRef.Name)
		link = interconnectv1alpha1.Link{
			ObjectMeta: metav1.ObjectMeta{
				Name: linkName,
				Labels: map[string]string{
					utils.InterconnectProjectIDLabel: networkLink.Labels[utils.NetworkProjectIDLabel],
				},
			},
			Spec: interconnectv1alpha1.LinkSpec{
				SourceClusterRef: corev1.ObjectReference{
					Name: sourceName,
				},
				TargetClusterRef: corev1.ObjectReference{
					Name: targetName,
				},
			},
		}

		if err := controllerutil.SetOwnerReference(networkLink, &link, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if err := c.Create(ctx, &link); err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err)
			return err
		}
	} else if !hasOwnerRef(networkLink, &link) {
		log.Infof("Network %s: Upgrading interconnect link %s [%s <-> %s]",
			networkLink.Spec.NetworkRef.Name, linkName, networkLink.Spec.SourceClusterRef.Name, networkLink.Spec.TargetClusterRef.Name)
		if err := controllerutil.SetOwnerReference(networkLink, &link, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if err := c.Update(ctx, &link); err != nil && !errors.IsNotFound(err) {
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

func (c *NetworkLinkController) unbindLink(ctx context.Context, networkLink *networkv1alpha1.NetworkLink) error {
	var link interconnectv1alpha1.Link
	_, _, linkName := newInterconnectLinkName(
		networkLink.Spec.SourceClusterRef,
		networkLink.Spec.TargetClusterRef)
	linkKey := client.ObjectKey{
		Name: linkName,
	}
	if err := c.Get(ctx, linkKey, &link); err == nil {
		if hasOwnerRef(networkLink, &link) {
			log.Infof("Network %s: Downgrading interconnect link %s [%s -> %s]",
				networkLink.Spec.NetworkRef.Name, linkName, networkLink.Spec.SourceClusterRef.Name, networkLink.Spec.TargetClusterRef.Name)
			if err := controllerutil.RemoveOwnerReference(networkLink, &link, c.Scheme); err != nil {
				log.Error(err)
				return err
			}

			if len(link.OwnerReferences) == 0 {
				if err := c.Delete(ctx, &link); err != nil && !errors.IsNotFound(err) {
					if errors.IsConflict(err) {
						log.Warn(err)
					} else {
						log.Error(err)
					}
					return err
				}
			} else {
				if err := c.Update(ctx, &link); err != nil && !errors.IsNotFound(err) {
					if errors.IsConflict(err) {
						log.Warn(err)
					} else {
						log.Error(err)
					}
					return err
				}
			}
		}
	}
	return nil
}

func newInterconnectLinkName(sourceClusterRef, targetClusterRef corev1.ObjectReference) (string, string, string) {
	clusterNames := sort.StringSlice{sourceClusterRef.Name, targetClusterRef.Name}
	clusterNames.Sort()
	sourceName := clusterNames[0]
	targetName := clusterNames[1]
	linkHash, err := hash(sourceName, targetName)
	if err != nil {
		log.Fatal(err)
		return "", "", ""
	}
	return sourceName, targetName, fmt.Sprintf("%s-%s", targetClusterRef.Name, linkHash)
}
