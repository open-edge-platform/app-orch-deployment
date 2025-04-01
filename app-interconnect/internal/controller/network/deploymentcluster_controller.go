// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"fmt"
	admv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

const deploymentClusterFinalizer = "deploymentcluster.network.app.edge-orchestrator.intel.com/finalizer"

func AddDeploymentClusterController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &DeploymentClusterController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// DeploymentClusterController is a Controller for propagating network resources to the interconnect control plane.
type DeploymentClusterController struct {
	*controller.Controller
}

func (c *DeploymentClusterController) Setup(mgr manager.Manager) error {
	log.Info("Setting up deploymentcluster-controller")
	controller, err := controllerruntime.New("deploymentcluster-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}

	// Watch Deployment resources and enqueue requests for each Deployment.
	err = controller.Watch(source.Kind(c.Cache,
		&admv1beta1.DeploymentCluster{},
		&handler.TypedEnqueueRequestForObject[*admv1beta1.DeploymentCluster]{}))
	if err != nil {
		log.Error(err)
		return err
	}

	// Watch Deployment resources and enqueue requests for each DeploymentCluster in each Deployment.
	err = controller.Watch(source.Kind(c.Cache,
		&admv1beta1.Deployment{},
		handler.TypedEnqueueRequestsFromMapFunc[*admv1beta1.Deployment, reconcile.Request](func(ctx context.Context, deployment *admv1beta1.Deployment) []reconcile.Request {
			var deploymentClusterList admv1beta1.DeploymentClusterList
			if err := c.List(ctx, &deploymentClusterList, client.MatchingFields{
				deploymentUIDField: string(deployment.UID),
			}); err != nil {
				log.Error(err)
				return nil
			}

			var requests []reconcile.Request
			for _, deploymentCluster := range deploymentClusterList.Items {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      deploymentCluster.Name,
						Namespace: deploymentCluster.Namespace,
					},
				})
			}
			return requests
		})))
	if err != nil {
		log.Error(err)
		return err
	}

	// Watch Network resources and enqueue requests for each DeploymentCluster in each Deployment in each Network.
	err = controller.Watch(source.Kind(c.Cache,
		&networkv1alpha1.Network{},
		handler.TypedEnqueueRequestsFromMapFunc[*networkv1alpha1.Network, reconcile.Request](func(ctx context.Context, network *networkv1alpha1.Network) []reconcile.Request {
			var deploymentList admv1beta1.DeploymentList
			if err := c.List(ctx, &deploymentList, client.MatchingFields{
				networkNameField: network.Name,
			}); err != nil {
				log.Error(err)
				return nil
			}

			var requests []reconcile.Request
			for _, deployment := range deploymentList.Items {
				var deploymentClusterList admv1beta1.DeploymentClusterList
				if err := c.List(ctx, &deploymentClusterList, client.MatchingFields{
					deploymentUIDField: string(deployment.UID),
				}); err != nil {
					log.Error(err)
					return nil
				}

				for _, deploymentCluster := range deploymentClusterList.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      deploymentCluster.Name,
							Namespace: deploymentCluster.Namespace,
						},
					})
				}
			}
			return requests
		}), predicate.TypedGenerationChangedPredicate[*networkv1alpha1.Network]{}))
	if err != nil {
		log.Error(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&networkv1alpha1.NetworkCluster{},
		handler.TypedEnqueueRequestsFromMapFunc[*networkv1alpha1.NetworkCluster, reconcile.Request](func(ctx context.Context, networkCluster *networkv1alpha1.NetworkCluster) []reconcile.Request {
			var deploymentClusterList admv1beta1.DeploymentClusterList
			if err := c.List(ctx, &deploymentClusterList,
				client.MatchingLabels{
					networkNameKey: networkCluster.Labels[networkNameKey],
				}, client.MatchingFields{
					clusterNamespaceField: networkCluster.Spec.ClusterRef.Namespace,
					clusterNameField:      networkCluster.Spec.ClusterRef.Name,
				}); err != nil {
				log.Error(err)
				return nil
			}

			var requests []reconcile.Request
			for _, deploymentCluster := range deploymentClusterList.Items {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      deploymentCluster.Name,
						Namespace: deploymentCluster.Namespace,
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

func (c *DeploymentClusterController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling DeploymentCluster %s", request.NamespacedName)

	// Get the DeploymentCluster that triggered reconciliation.
	var deploymentCluster admv1beta1.DeploymentCluster
	deploymentClusterKey := request.NamespacedName
	if err := c.Get(ctx, deploymentClusterKey, &deploymentCluster); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// If the Deployment does not specify a network reference, skip reconciliation.
	networkName := deploymentCluster.Labels[networkNameKey]
	if networkName == "" {
		log.Infof("DeploymentCluster %s is not labeled with a valid Network; skipping reconciliation...", deploymentCluster.Name)
		return reconcile.Result{}, nil
	}

	var networkCluster networkv1alpha1.NetworkCluster
	networkClusterKey := client.ObjectKey{
		Name: newNetworkClusterName(
			corev1.ObjectReference{
				Name: networkName,
			},
			corev1.ObjectReference{
				Namespace: deploymentCluster.Spec.Namespace,
				Name:      deploymentCluster.Spec.ClusterID,
			}),
	}
	if err := c.Get(ctx, networkClusterKey, &networkCluster); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}

		// If the DeploymentCluster is being deleted, unbind it from the Network and remove the finalizers.
		if deploymentCluster.DeletionTimestamp != nil {
			if controllerutil.RemoveFinalizer(&deploymentCluster, deploymentClusterFinalizer) {
				log.Infof("Removing finalizer %s from DeploymentCluster %s", deploymentClusterFinalizer, deploymentCluster.Name)
				if err := c.Update(ctx, &deploymentCluster); err != nil && !errors.IsNotFound(err) {
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

		// Add the deployment finalizer to the DeploymentCluster if necessary.
		if controllerutil.AddFinalizer(&deploymentCluster, deploymentClusterFinalizer) {
			log.Infof("Adding finalizer %s to DeploymentCluster %s", deploymentClusterFinalizer, deploymentCluster.Name)
			if err := c.Update(ctx, &deploymentCluster); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return reconcile.Result{}, err
			}
		}

		// Get the Network associated with the Deployment.
		var network networkv1alpha1.Network
		networkKey := client.ObjectKey{
			Name: networkName,
		}
		if err := c.Get(ctx, networkKey, &network); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
			log.Warnf("Network %s not found for DeploymentCluster %s", networkName, deploymentCluster.Name)
			return reconcile.Result{}, nil
		}

		networkCluster = networkv1alpha1.NetworkCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: networkClusterKey.Name,
				Labels: map[string]string{
					networkNameKey:              network.Name,
					utils.NetworkProjectIDLabel: network.Labels[utils.NetworkProjectIDLabel],
				},
			},
			Spec: networkv1alpha1.NetworkClusterSpec{
				// TODO: Update the Network reference to point to the Nexus resource once it's integrated.
				NetworkRef: corev1.ObjectReference{
					Name: network.Name,
				},
				ClusterRef: corev1.ObjectReference{
					// TODO: Should we populate this reference with the actual Cluster metadata (UID etc)?
					Namespace: deploymentCluster.Spec.Namespace,
					Name:      deploymentCluster.Spec.ClusterID,
				},
			},
		}

		if err := controllerutil.SetOwnerReference(&network, &networkCluster, c.Scheme); err != nil {
			log.Error(err)
			return reconcile.Result{}, err
		}

		if err := c.Create(ctx, &networkCluster); err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
	}

	// If the DeploymentCluster is being deleted, unbind it from the Network and remove the finalizers.
	if deploymentCluster.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(&deploymentCluster, deploymentClusterFinalizer) {
			var deploymentClusterRefs []corev1.ObjectReference
			for _, deploymentClusterRef := range networkCluster.Status.DeploymentClusterRefs {
				if deploymentClusterRef.UID != deploymentCluster.UID {
					deploymentClusterRefs = append(deploymentClusterRefs, deploymentClusterRef)
				}
			}

			if len(deploymentClusterRefs) == 0 {
				if err := c.Delete(ctx, &networkCluster); err != nil && !errors.IsNotFound(err) {
					if errors.IsConflict(err) {
						log.Warn(err)
					} else {
						log.Error(err)
					}
					return reconcile.Result{}, err
				}
			} else if len(deploymentClusterRefs) != len(networkCluster.Status.DeploymentClusterRefs) {
				networkCluster.Status.DeploymentClusterRefs = deploymentClusterRefs
				if err := c.Status().Update(ctx, &networkCluster); err != nil && !errors.IsNotFound(err) {
					if errors.IsConflict(err) {
						log.Warn(err)
					} else {
						log.Error(err)
					}
					return reconcile.Result{}, err
				}
			}

			log.Infof("Removing finalizer %s from DeploymentCluster %s", deploymentClusterFinalizer, deploymentCluster.Name)
			if err := c.Update(ctx, &deploymentCluster); err != nil && !errors.IsNotFound(err) {
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

	// Iterate through existing DeploymentCluster references in the NetworkCluster and determine whether
	// this DeploymentCluster is already referenced. If not, add it.
	for _, deploymentClusterRef := range networkCluster.Status.DeploymentClusterRefs {
		if deploymentClusterRef.UID == deploymentCluster.UID {
			return reconcile.Result{}, nil
		}
	}

	networkCluster.Status.DeploymentClusterRefs = append(networkCluster.Status.DeploymentClusterRefs, corev1.ObjectReference{
		APIVersion: deploymentCluster.APIVersion,
		Kind:       deploymentCluster.Kind,
		Namespace:  deploymentCluster.Namespace,
		Name:       deploymentCluster.Name,
		UID:        deploymentCluster.UID,
	})
	if err := c.Status().Update(ctx, &networkCluster); err != nil && !errors.IsNotFound(err) {
		if errors.IsConflict(err) {
			log.Warn(err)
		} else {
			log.Error(err)
		}
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func newNetworkClusterName(networkRef, clusterRef corev1.ObjectReference) string {
	return fmt.Sprintf("%s-%s", networkRef.Name, clusterRef.Name)
}
