// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
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

const deploymentFinalizer = "deployment.network.app.edge-orchestrator.intel.com/finalizer"

func AddDeploymentController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &DeploymentController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// DeploymentController is a Controller for propagating network resources to the interconnect control plane.
type DeploymentController struct {
	*controller.Controller
}

func (c *DeploymentController) Setup(mgr manager.Manager) error {
	log.Info("Setting up deployment-controller")
	controller, err := controllerruntime.New("deployment-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}

	// Watch Deployment resources and enqueue requests for each Deployment.
	err = controller.Watch(source.Kind(c.Cache,
		&admv1beta1.Deployment{},
		&handler.TypedEnqueueRequestForObject[*admv1beta1.Deployment]{}))
	if err != nil {
		log.Error(err)
		return err
	}

	// Watch DeploymentCluster resources and enqueue a request for the reference resource.
	err = controller.Watch(source.Kind(c.Cache,
		&admv1beta1.DeploymentCluster{},
		handler.TypedEnqueueRequestsFromMapFunc[*admv1beta1.DeploymentCluster, reconcile.Request](func(ctx context.Context, deploymentCluster *admv1beta1.DeploymentCluster) []reconcile.Request {
			var deploymentList admv1beta1.DeploymentList
			if err := c.List(ctx, &deploymentList, client.MatchingFields{
				deploymentUIDField: deploymentCluster.Spec.DeploymentID,
			}); err != nil {
				log.Error(err)
				return nil
			}

			var requests []reconcile.Request
			for _, deployment := range deploymentList.Items {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      deployment.Name,
						Namespace: deployment.Namespace,
					},
				})
			}
			return requests
		}), predicate.TypedGenerationChangedPredicate[*admv1beta1.DeploymentCluster]{}))
	if err != nil {
		log.Error(err)
		return err
	}

	// Watch Network resources and enqueue Deployment requests for each Network.
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
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      deployment.Name,
						Namespace: deployment.Namespace,
					},
				})
			}
			return requests
		}), predicate.TypedResourceVersionChangedPredicate[*networkv1alpha1.Network]{}))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *DeploymentController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling Deployment %s", request.NamespacedName)

	// Get the Deployment that triggered reconciliation.
	var deployment admv1beta1.Deployment
	if err := c.Get(ctx, request.NamespacedName, &deployment); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// If the Deployment is being deleted, perform finalization by unbinding the DeploymentCluster from the network.
	if deployment.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(&deployment, deploymentFinalizer) {
			if err := c.unbindDeployment(ctx, &deployment); err != nil {
				return reconcile.Result{}, err
			}

			log.Infof("Removing finalizer %s from Deployment %s", deploymentFinalizer, deployment.Name)
			if err := c.Update(ctx, &deployment); err != nil && !errors.IsNotFound(err) {
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

	// Label the DeploymentClusters associated with the Deployment for easy lookup in other controllers.
	var deploymentClusterList admv1beta1.DeploymentClusterList
	if err := c.List(ctx, &deploymentClusterList, client.MatchingLabels{
		deploymentIDKey: string(deployment.UID),
	}); err != nil {
		log.Error(err)
		return reconcile.Result{}, err
	}

	for _, deploymentCluster := range deploymentClusterList.Items {
		if updateLabels(&deploymentCluster, map[string]string{
			deploymentNameKey:      deployment.Name,
			deploymentNamespaceKey: deployment.Namespace,
			networkNameKey:         deployment.Spec.NetworkRef.Name,
		}) {
			log.Infof("Labeling DeploymentCluster %s for Deployment %s", deploymentCluster.Name, deployment.Name)
			if err := c.Update(ctx, &deploymentCluster); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return reconcile.Result{}, err
			}
		}
	}

	// If the Deployment does not specify a network reference, do not create a Network.
	if deployment.Spec.NetworkRef.Name == "" {
		// If a finalizer has already been added to the Deployment, ensure
		log.Infof("Skipping reconciliation of Deployment %s: network reference is empty", deployment.Name)
		return reconcile.Result{}, nil
	}

	// Add the deployment finalizer to the Deployment if necessary.
	if controllerutil.AddFinalizer(&deployment, deploymentFinalizer) {
		log.Infof("Adding finalizer %s to Deployment %s", deploymentFinalizer, deployment.Name)
		if err := c.Update(ctx, &deployment); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Bind the deployment to the Network.
	if err := c.bindDeployment(ctx, &deployment); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (c *DeploymentController) bindDeployment(ctx context.Context, deployment *admv1beta1.Deployment) error {
	var network networkv1alpha1.Network
	networkKey := client.ObjectKey{
		Name: deployment.Spec.NetworkRef.Name,
	}
	if err := c.Get(ctx, networkKey, &network); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}

		network = networkv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name: deployment.Spec.NetworkRef.Name,
				Labels: map[string]string{
					networkNameKey:              deployment.Labels[networkNameKey],
					utils.NetworkProjectIDLabel: deployment.Labels[utils.AppOrchProjectIDLabel],
				},
			},
			Spec: networkv1alpha1.NetworkSpec{
				// TODO: Populate the Network spec with a reference to the Nexus Network resource.
			},
		}

		if err := c.Create(ctx, &network); err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err)
			return err
		}
		return nil
	}

	for _, deploymentRef := range network.Status.DeploymentRefs {
		if deploymentRef.UID == deployment.UID {
			return nil
		}
	}

	network.Status.DeploymentRefs = append(network.Status.DeploymentRefs, corev1.ObjectReference{
		APIVersion: deployment.APIVersion,
		Kind:       deployment.Kind,
		Namespace:  deployment.Namespace,
		Name:       deployment.Name,
		UID:        deployment.UID,
	})
	if err := c.Status().Update(ctx, &network); err != nil && !errors.IsNotFound(err) {
		if errors.IsConflict(err) {
			log.Warn(err)
		} else {
			log.Error(err)
		}
		return err
	}
	return nil
}

func (c *DeploymentController) unbindDeployment(ctx context.Context, deployment *admv1beta1.Deployment) error {
	var network networkv1alpha1.Network
	networkKey := client.ObjectKey{
		Name: deployment.Spec.NetworkRef.Name,
	}
	if err := c.Get(ctx, networkKey, &network); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}
		return nil
	}

	var deploymentRefs []corev1.ObjectReference
	for _, deploymentRef := range network.Status.DeploymentRefs {
		if deploymentRef.UID != deployment.UID {
			deploymentRefs = append(deploymentRefs, deploymentRef)
		}
	}

	if len(deploymentRefs) == 0 {
		if err := c.Delete(ctx, &network); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return err
		}
	} else if len(deploymentRefs) != len(network.Status.DeploymentRefs) {
		network.Status.DeploymentRefs = deploymentRefs
		if err := c.Status().Update(ctx, &network); err != nil && !errors.IsNotFound(err) {
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
