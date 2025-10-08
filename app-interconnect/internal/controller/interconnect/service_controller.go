// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interconnect

import (
	"context"
	"fmt"
	"strings"
	"time"

	clusterclient "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/utils"
	skupperlib "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper"
	skupperutils "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/utils/skupper"
	interconnectv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/interconnect/v1alpha1"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/workqueue"
	controllerruntime "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const serviceFinalizer = "service.interconnect.app.edge-orchestrator.intel.com/finalizer"

func AddServiceController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &ServiceController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// ServiceController is a Controller for setting up the Service
type ServiceController struct {
	*controller.Controller
}

func (c *ServiceController) Setup(mgr manager.Manager) error {
	log.Info("Setting up service-controller")
	controller, err := controllerruntime.New("service-controller", mgr, controllerruntime.Options{
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

func (c *ServiceController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling Service %s", request.NamespacedName)
	var service interconnectv1alpha1.Service
	if err := c.Get(ctx, request.NamespacedName, &service); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if service.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(&service, serviceFinalizer) {
			if service.Status.Phase != interconnectv1alpha1.ServiceUnexposing {
				service.Status.Phase = interconnectv1alpha1.ServiceUnexposing
				if err := c.Status().Update(ctx, &service); err != nil {
					if !errors.IsNotFound(err) {
						log.Error(err)
						return reconcile.Result{}, err
					}
				}
				return reconcile.Result{}, nil
			}
		}
	} else if !controllerutil.ContainsFinalizer(&service, serviceFinalizer) {
		controllerutil.AddFinalizer(&service, serviceFinalizer)
		if err := c.Update(ctx, &service); err != nil {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}
	serviceName := service.Spec.ServiceRef.Name
	clusterID := clusterclient.ClusterID(service.Spec.ClusterRef.Name)
	projectID := clusterclient.ProjectID(service.Labels[utils.InterconnectProjectIDLabel])

	clusterConfig, err := c.Clusters.GetClusterConfig(ctx, clusterID, projectID)
	if err != nil {
		log.Warn(err)
		return reconcile.Result{}, err
	}

	switch service.Status.Phase {
	case interconnectv1alpha1.ServicePending:
		// Move to the Configuring phase
		service.Status.Phase = interconnectv1alpha1.ServiceExposing
		if err := c.Status().Update(ctx, &service); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
		}
	case interconnectv1alpha1.ServiceExposing:
		ports := make([]int, 0)
		for _, port := range service.Spec.ExposePorts {
			ports = append(ports, int(port.Port))
		}

		var targetName string
		if service.Spec.ServiceRef.Namespace != skupperutils.DefaultSkupperNamespace {
			targetName = fmt.Sprintf("%s.%s", service.Spec.ServiceRef.Name, service.Spec.ServiceRef.Namespace)
		} else {
			targetName = service.Spec.ServiceRef.Name
		}
		err = skupperlib.SkupperExposeService(ctx, clusterConfig, targetName, "service", serviceName, ports)
		if err != nil {
			log.Warn(err)
			return reconcile.Result{}, err
		}
		service.Status.Phase = interconnectv1alpha1.ServiceExposed
		if err := c.Status().Update(ctx, &service); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
		}
	case interconnectv1alpha1.ServiceExposed:
		log.Infow("Service is already exposed", dazl.String("service", serviceName),
			dazl.String("cluster", string(clusterID)))
	case interconnectv1alpha1.ServiceUnexposing:
		err = skupperlib.SkupperUnexposeService(ctx, clusterConfig, serviceName, "service", service.Name)
		if err == nil || (err != nil && strings.Contains(err.Error(), "No skupper service interfaces defined")) {
			controllerutil.RemoveFinalizer(&service, serviceFinalizer)
			if err := c.Update(ctx, &service); err != nil {
				if !errors.IsNotFound(err) && !errors.IsConflict(err) {
					log.Error(err)
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, nil
			}
		} else {
			log.Warn(err)
			return reconcile.Result{}, err
		}

	}
	return reconcile.Result{}, nil
}
