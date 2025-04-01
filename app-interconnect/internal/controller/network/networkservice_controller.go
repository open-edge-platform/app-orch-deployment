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
	"time"
)

const networkServiceFinalizer = "networkservice.network.app.edge-orchestrator.intel.com/finalizer"

func AddNetworkServiceController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &NetworkServiceController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// Ignore stuttering type name due to multiple controllers being in the same package.
//revive:disable

// NetworkServiceController is a Controller for setting up the NetworkService
type NetworkServiceController struct {
	*controller.Controller
}

//revive:enable

func (c *NetworkServiceController) Setup(mgr manager.Manager) error {
	log.Info("Setting up networkservice-controller")
	controller, err := controllerruntime.New("networkservice-controller", mgr, controllerruntime.Options{
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
		&interconnectv1alpha1.Service{},
		handler.TypedEnqueueRequestForOwner[*interconnectv1alpha1.Service](
			c.Scheme, c.RESTMapper(), &networkv1alpha1.NetworkService{})))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *NetworkServiceController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling NetworkService %s", request.NamespacedName)
	var networkService networkv1alpha1.NetworkService
	if err := c.Get(ctx, request.NamespacedName, &networkService); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if networkService.DeletionTimestamp != nil {
		if controllerutil.RemoveFinalizer(&networkService, networkServiceFinalizer) {
			if err := c.removeInterconnectService(ctx, &networkService); err != nil {
				return reconcile.Result{}, err
			}

			log.Infof("Removing finalizer %s from NetworkService %s", networkServiceFinalizer, networkService.Name)
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

	if controllerutil.AddFinalizer(&networkService, networkServiceFinalizer) {
		log.Infof("Adding finalizer %s to NetworkService %s", networkServiceFinalizer, networkService.Name)
		if err := c.Update(ctx, &networkService); err != nil && !errors.IsNotFound(err) {
			if errors.IsConflict(err) {
				log.Warn(err)
			} else {
				log.Error(err)
			}
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if err := c.addInterconnectService(ctx, &networkService); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (c *NetworkServiceController) addInterconnectService(ctx context.Context, networkService *networkv1alpha1.NetworkService) error {
	var service interconnectv1alpha1.Service
	serviceKey := client.ObjectKey{
		Name: newInterconnectServiceName(networkService.Spec.ClusterRef, networkService.Spec.ServiceRef),
	}
	if err := c.Get(ctx, serviceKey, &service); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}

		log.Infof("Network %s: Creating interconnect Service %s",
			networkService.Spec.NetworkRef.Name, serviceKey.Name)

		exposePorts := make([]interconnectv1alpha1.ServicePort, 0, len(networkService.Spec.ExposePorts))
		for _, exposePort := range networkService.Spec.ExposePorts {
			exposePorts = append(exposePorts, interconnectv1alpha1.ServicePort{
				Port: exposePort.Port,
			})
		}

		service = interconnectv1alpha1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceKey.Name,
				Labels: map[string]string{
					networkNameKey:                   networkService.Labels[networkNameKey],
					utils.InterconnectProjectIDLabel: networkService.Labels[utils.NetworkProjectIDLabel],
				},
			},
			Spec: interconnectv1alpha1.ServiceSpec{
				ClusterRef:  networkService.Spec.ClusterRef,
				ServiceRef:  networkService.Spec.ServiceRef,
				ExposePorts: exposePorts,
			},
		}

		log.Infof("Network %s: Binding NetworkService %s to interconnect Service %s",
			networkService.Spec.NetworkRef.Name, networkService.Name, serviceKey.Name)
		if err := controllerutil.SetOwnerReference(networkService, &service, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if err := c.Create(ctx, &service); err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err)
				return err
			}
			return nil
		}
		return nil
	}

	if !hasOwnerRef(&service, networkService) {
		log.Infof("Network %s: Binding NetworkService %s to interconnect Service %s",
			networkService.Spec.NetworkRef.Name, networkService.Name, serviceKey.Name)
		if err := controllerutil.SetOwnerReference(networkService, &service, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		log.Infof("Network %s: Updating interconnect Service %s",
			networkService.Spec.NetworkRef.Name, serviceKey.Name)
		if err := c.Update(ctx, &service); err != nil && !errors.IsNotFound(err) {
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

func (c *NetworkServiceController) removeInterconnectService(ctx context.Context, networkService *networkv1alpha1.NetworkService) error {
	var service interconnectv1alpha1.Service
	serviceKey := client.ObjectKey{
		Name: newInterconnectServiceName(networkService.Spec.ClusterRef, networkService.Spec.ServiceRef),
	}
	if err := c.Get(ctx, serviceKey, &service); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return err
		}
		return nil
	}

	if hasOwnerRef(&service, networkService) {
		log.Infof("Network %s: Unbinding NetworkService %s from interconnect Service %s",
			networkService.Spec.NetworkRef.Name, networkService.Name, serviceKey.Name)
		if err := controllerutil.RemoveOwnerReference(networkService, &service, c.Scheme); err != nil {
			log.Error(err)
			return err
		}

		if len(service.OwnerReferences) == 0 {
			log.Infof("Network %s: Deleting interconnect Service %s",
				networkService.Spec.NetworkRef.Name, serviceKey.Name)
			if err := c.Delete(ctx, &service); err != nil && !errors.IsNotFound(err) {
				if errors.IsConflict(err) {
					log.Warn(err)
				} else {
					log.Error(err)
				}
				return err
			}
		} else {
			log.Infof("Network %s: Updating interconnect Service %s",
				networkService.Spec.NetworkRef.Name, serviceKey.Name)
			if err := c.Update(ctx, &service); err != nil && !errors.IsNotFound(err) {
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

func newInterconnectServiceName(clusterRef corev1.ObjectReference, serviceRef corev1.ObjectReference) string {
	serviceHash, err := hash(clusterRef.Namespace, serviceRef.Name)
	if err != nil {
		log.Fatal(err)
		return ""
	}
	return fmt.Sprintf("%s-%s", clusterRef.Name, serviceHash)
}
