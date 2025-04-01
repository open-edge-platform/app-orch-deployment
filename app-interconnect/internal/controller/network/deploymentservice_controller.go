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
	"k8s.io/client-go/kubernetes"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerruntime "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"strings"
	"time"
)

const deploymentServiceFinalizer = "deploymentservice.network.app.edge-orchestrator.intel.com/finalizer"

func AddDeploymentServiceController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &DeploymentServiceController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// DeploymentServiceController is a Controller for propagating network resources to the interconnect control plane.
type DeploymentServiceController struct {
	*controller.Controller
}

func (c *DeploymentServiceController) Setup(mgr manager.Manager) error {
	log.Info("Setting up deploymentservice-controller")
	controller, err := controllerruntime.New("deploymentservice-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&admv1beta1.DeploymentCluster{},
		&handler.TypedEnqueueRequestForObject[*admv1beta1.DeploymentCluster]{}))
	if err != nil {
		log.Error(err)
		return err
	}

	err = controller.Watch(source.Kind(c.Cache,
		&admv1beta1.Deployment{},
		handler.TypedEnqueueRequestsFromMapFunc[*admv1beta1.Deployment, reconcile.Request](func(ctx context.Context, deployment *admv1beta1.Deployment) []reconcile.Request {
			var deploymentClusterList admv1beta1.DeploymentClusterList
			matchLabels := map[string]string{
				deploymentIDKey: string(deployment.UID),
			}
			if err := c.List(ctx, &deploymentClusterList, client.MatchingLabels(matchLabels)); err != nil {
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

	err = controller.Watch(newDeploymentServiceSource(c.Clusters, c.Cache,
		&handler.TypedEnqueueRequestForObject[*admv1beta1.DeploymentCluster]{}))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *DeploymentServiceController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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

	// If the DeploymentCluster is being deleted, unbind its services from the network and remove the finalizers.
	if deploymentCluster.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(&deploymentCluster, deploymentServiceFinalizer) {
			if err := c.unbindServices(ctx, &deploymentCluster); err != nil {
				return reconcile.Result{}, err
			}

			log.Infof("Removing finalizer %s from DeploymentCluster %s", deploymentServiceFinalizer, deploymentCluster.Name)
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

	// If neither the Deployment nor the DeploymentCluster is being deleted, ensure the service finalizer has
	// been added to both objects. Details of the specs of both are required to bind and unbind of clusters.
	if controllerutil.AddFinalizer(&deploymentCluster, deploymentServiceFinalizer) {
		log.Infof("Adding finalizer %s to DeploymentCluster %s", deploymentServiceFinalizer, deploymentCluster.Name)
		if err := c.Update(ctx, &deploymentCluster); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if err := c.bindServices(ctx, &deploymentCluster); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (c *DeploymentServiceController) bindServices(ctx context.Context, deploymentCluster *admv1beta1.DeploymentCluster) error {
	networkName := deploymentCluster.Labels[networkNameKey]
	if networkName == "" {
		log.Warnf("Cannot bind services for DeploymentCluster %s: %s not found", deploymentCluster.Name, networkNameKey)
		return nil
	}

	log.Infof("Found %s: %s", networkNameKey, networkName)
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
			return err
		}
		log.Warnf("Skipping service binding for %s: waiting for NetworkCluster %s", deploymentCluster.Name, networkClusterKey.Name)
		return nil
	}

	// TODO: Get the project ID from resource labels.
	var projectID clusterclient.ProjectID
	clusterID := clusterclient.ClusterID(deploymentCluster.Spec.ClusterID)

	clusterConfig, err := c.Clusters.GetClusterConfig(ctx, clusterID, projectID)
	if err != nil {
		log.Error(err)
		return err
	}

	clusterClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Error(err)
		return err
	}

	// Get the set of app IDs in the cluster.
	deployedHelmReleaseNames := make(map[string]bool)
	for _, app := range deploymentCluster.Status.Apps {
		deployedHelmReleaseNames[app.Id] = true
	}

	// List all the Services matching the deployment's apps and bind the ones that are annotated for export.
	services := clusterClient.CoreV1().Services(corev1.NamespaceAll)
	serviceList, err := services.List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err)
		return nil
	}
	for _, service := range serviceList.Items {
		if appID, ok := service.Annotations[helmReleaseNameAnnotation]; ok && deployedHelmReleaseNames[appID] && service.Annotations[networkExposeServiceAnnotation] == "true" {
			if err := c.bindService(ctx, &networkCluster, services, &service); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *DeploymentServiceController) bindService(ctx context.Context, networkCluster *networkv1alpha1.NetworkCluster, services clientcorev1.ServiceInterface, service *corev1.Service) error {
	networkServiceKey := client.ObjectKey{
		Name: newNetworkServiceName(
			networkCluster.Spec.NetworkRef,
			networkCluster.Spec.ClusterRef,
			corev1.ObjectReference{
				Namespace: service.Namespace,
				Name:      service.Name,
			}),
	}

	var networkService networkv1alpha1.NetworkService
	if err := c.Get(ctx, networkServiceKey, &networkService); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}

		// If the edge service has been marked for deletion, just remove the finalizer to unblock it.
		// We don't need to do anything since the NetworkService doesn't exist.
		if service.DeletionTimestamp != nil {
			if controllerutil.RemoveFinalizer(service, deploymentServiceFinalizer) {
				log.Infof("Removing finalizer %s from Service %s", deploymentServiceFinalizer, service.Name)
				if _, err := services.Update(ctx, service, metav1.UpdateOptions{}); err != nil && !errors.IsNotFound(err) {
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

		// If the service is not being deleted and the finalizer is not present, add the finalizer and create
		// a new NetworkService.
		if controllerutil.AddFinalizer(service, deploymentServiceFinalizer) {
			log.Infof("Adding finalizer %s to Service %s", deploymentServiceFinalizer, service.Name)
			if _, err := services.Update(ctx, service, metav1.UpdateOptions{}); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return err
			}
		}

		log.Infof("Creating NetworkService %s for %s", service.Name, networkServiceKey.Name)
		portStrings := strings.Split(service.Annotations[networkExposePortsAnnotation], ",")
		portStrings = append(portStrings, service.Annotations[networkExposePortAnnotation])

		var exposePorts []networkv1alpha1.NetworkServicePort
		for _, portString := range portStrings {
			if portString != "" {
				port, err := strconv.ParseUint(portString, 10, 16)
				if err != nil {
					log.Error(err)
					return nil
				}
				exposePorts = append(exposePorts, networkv1alpha1.NetworkServicePort{
					Port: int32(uint16(port)),
				})
			}
		}

		// If no ports were configured explicitly in annotations, expose all ports in the service.
		if len(exposePorts) == 0 {
			for _, port := range service.Spec.Ports {
				exposePorts = append(exposePorts, networkv1alpha1.NetworkServicePort{
					Port: port.Port,
				})
			}
		}

		networkService = networkv1alpha1.NetworkService{
			ObjectMeta: metav1.ObjectMeta{
				Name: networkServiceKey.Name,
				Labels: map[string]string{
					networkNameKey:              networkCluster.Spec.NetworkRef.Name,
					clusterNamespaceKey:         networkCluster.Spec.ClusterRef.Namespace,
					clusterNameKey:              networkCluster.Spec.ClusterRef.Name,
					utils.NetworkProjectIDLabel: networkCluster.Labels[utils.NetworkProjectIDLabel],
				},
			},
			Spec: networkv1alpha1.NetworkServiceSpec{
				NetworkRef: networkCluster.Spec.NetworkRef,
				ClusterRef: networkCluster.Spec.ClusterRef,
				ServiceRef: corev1.ObjectReference{
					Namespace: service.Namespace,
					Name:      service.Name,
				},
				ExposePorts: exposePorts,
			},
		}

		if err := controllerutil.SetOwnerReference(networkCluster, &networkService, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if err := c.Create(ctx, &networkService); err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err)
			return err
		}
		return nil
	}

	// If the edge service has been marked for deletion and the finalizer is still present, delete the NetworkService
	// and remove the finalizer from the edge service.
	if service.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(service, deploymentServiceFinalizer) {
			log.Infof("Deleting NetworkService %s for %s", networkServiceKey.Name, service.Name)
			if err := c.Delete(ctx, &networkService); err != nil && !errors.IsNotFound(err) {
				if !errors.IsConflict(err) {
					log.Error(err)
				}
				return err
			}

			log.Infof("Removing finalizer %s from Service %s", deploymentServiceFinalizer, service.Name)
			if _, err := services.Update(ctx, service, metav1.UpdateOptions{}); err != nil && !errors.IsNotFound(err) {
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

	for _, networkServiceRef := range networkCluster.Status.Services {
		if networkServiceRef.Name == networkService.Name {
			return nil
		}
	}

	// If the NetworkService is not found in the NetworkCluster status, add it.
	log.Infof("Adding NetworkService %s to NetworkCluster %s", networkService.Name, networkCluster.Name)
	networkCluster.Status.Services = append(networkCluster.Status.Services, corev1.LocalObjectReference{
		Name: networkService.Name,
	})
	if err := c.Status().Update(ctx, networkCluster); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *DeploymentServiceController) unbindServices(ctx context.Context, deploymentCluster *admv1beta1.DeploymentCluster) error {
	var networkServiceList networkv1alpha1.NetworkServiceList
	if err := c.List(ctx, &networkServiceList, client.MatchingLabels{
		networkNameKey:      deploymentCluster.Labels[networkNameKey],
		clusterNamespaceKey: deploymentCluster.Spec.Namespace,
		clusterNameKey:      deploymentCluster.Spec.ClusterID,
	}); err != nil {
		log.Error(err)
		return err
	}

	for _, networkService := range networkServiceList.Items {
		log.Infof("Deleting NetworkService %s", networkService.Name)
		if err := c.Delete(ctx, &networkService); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func newNetworkServiceName(networkRef, clusterRef, serviceRef corev1.ObjectReference) string {
	networkServiceHash, err := hash(clusterRef.Name, serviceRef.Namespace, serviceRef.Name)
	if err != nil {
		log.Fatal(err)
		return ""
	}
	return fmt.Sprintf("%s-%s", networkRef.Name, networkServiceHash)
}
