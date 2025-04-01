// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interconnect

import (
	"context"
	clusterclient "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/utils"
	skupperlib "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper"
	interconnectv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/interconnect/v1alpha1"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	controllerruntime "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

const linkFinalizer = "link.interconnect.app.edge-orchestrator.intel.com/finalizer"

func AddLinkController(mgr manager.Manager, clusters clusterclient.Client) error {
	ctrl := &LinkController{
		Controller: controller.New(mgr, clusters),
	}
	return ctrl.Setup(mgr)
}

// LinkController is a Controller for setting up the Link
type LinkController struct {
	*controller.Controller
}

func (c *LinkController) Setup(mgr manager.Manager) error {
	log.Info("Setting up link-controller")
	controller, err := controllerruntime.New("link-controller", mgr, controllerruntime.Options{
		Reconciler:  c,
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Millisecond*10, time.Second*5),
	})
	if err != nil {
		log.Warn(err)
		return err
	}
	err = controller.Watch(source.Kind(c.Cache,
		&interconnectv1alpha1.Link{},
		&handler.TypedEnqueueRequestForObject[*interconnectv1alpha1.Link]{}))
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *LinkController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling Link %s", request.NamespacedName)
	var link interconnectv1alpha1.Link
	if err := c.Get(ctx, request.NamespacedName, &link); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}
	projectID := clusterclient.ProjectID(link.Labels[utils.InterconnectProjectIDLabel])

	if link.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(&link, linkFinalizer) {
			if link.Status.Phase != interconnectv1alpha1.LinkUnlinking {
				link.Status.Phase = interconnectv1alpha1.LinkUnlinking
				if err := c.Status().Update(ctx, &link); err != nil {
					if !errors.IsNotFound(err) {
						log.Error(err)
						return reconcile.Result{}, err
					}
				}
				return reconcile.Result{}, nil
			}
		}
	} else if !controllerutil.ContainsFinalizer(&link, linkFinalizer) {
		log.Infof("Adding finalizer %s to Link %s", linkFinalizer, link.Name)
		controllerutil.AddFinalizer(&link, linkFinalizer)
		if err := c.Update(ctx, &link); err != nil {
			log.Error(err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	log.Infow("Reconciling Link", dazl.String("source", link.Spec.SourceClusterRef.Name),
		dazl.String("target", link.Spec.TargetClusterRef.Name),
		dazl.String("phase", string(link.Status.Phase)))
	switch link.Status.Phase {
	case interconnectv1alpha1.LinkPending:
		// Move to the Configuring phase
		link.Status.Phase = interconnectv1alpha1.LinkLinking
		if err := c.Status().Update(ctx, &link); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
		}
	case interconnectv1alpha1.LinkLinking:

		sourceClusterID := clusterclient.ClusterID(link.Spec.SourceClusterRef.Name)
		sourceClusterConfig, err := c.Clusters.GetClusterConfig(ctx, sourceClusterID, projectID)
		if err != nil {
			log.Warn(err)
			return reconcile.Result{}, err
		}

		targetClusterID := clusterclient.ClusterID(link.Spec.TargetClusterRef.Name)
		targetClusterConfig, err := c.Clusters.GetClusterConfig(ctx, targetClusterID, projectID)
		if err != nil {
			log.Warn(err)
			return reconcile.Result{}, err
		}

		var sourceCluster interconnectv1alpha1.Cluster
		sourceClusterKey := types.NamespacedName{
			Name: link.Spec.SourceClusterRef.Name,
		}
		if err := c.Get(ctx, sourceClusterKey, &sourceCluster); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}

		var targetCluster interconnectv1alpha1.Cluster
		targetClusterClusterKey := types.NamespacedName{
			Name: link.Spec.TargetClusterRef.Name,
		}
		if err := c.Get(ctx, targetClusterClusterKey, &targetCluster); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}

		if sourceCluster.Status.Ingress == interconnectv1alpha1.IngressLoadBalancer {
			tokenSecret, err := skupperlib.SkupperTokenCreate(ctx, sourceClusterConfig, link.Name)
			if err != nil {
				return reconcile.Result{}, err
			}

			err = skupperlib.SkupperLinkCreate(ctx, targetClusterConfig, tokenSecret, link.Name)
			if err != nil {
				log.Warn(err)
				return reconcile.Result{}, err
			}
		} else if targetCluster.Status.Ingress == interconnectv1alpha1.IngressLoadBalancer {
			tokenSecret, err := skupperlib.SkupperTokenCreate(ctx, targetClusterConfig, link.Name)
			if err != nil {
				return reconcile.Result{}, err
			}

			err = skupperlib.SkupperLinkCreate(ctx, sourceClusterConfig, tokenSecret, link.Name)
			if err != nil {
				log.Warn(err)
				return reconcile.Result{}, err
			}
		}

		link.Status.Phase = interconnectv1alpha1.LinkLinked
		if err := c.Status().Update(ctx, &link); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err)
				return reconcile.Result{}, err
			}
		}
	case interconnectv1alpha1.LinkLinked:
		// Do nothing. The link is already up.
	case interconnectv1alpha1.LinkUnlinking:

		sourceClusterID := clusterclient.ClusterID(link.Spec.SourceClusterRef.Name)
		sourceClusterConfig, err := c.Clusters.GetClusterConfig(ctx, sourceClusterID, projectID)
		if err != nil {
			log.Warn(err)
			return reconcile.Result{}, err
		}

		targetClusterID := clusterclient.ClusterID(link.Spec.TargetClusterRef.Name)
		targetClusterConfig, err := c.Clusters.GetClusterConfig(ctx, targetClusterID, projectID)
		if err != nil {
			log.Warn(err)
			return reconcile.Result{}, err
		}

		err = skupperlib.SkupperLinkDelete(ctx, sourceClusterConfig, targetClusterConfig, link.Name)
		if err == nil || (err != nil && strings.Contains(err.Error(), "No such link")) {
			log.Infof("Removing finalizer %s from Link %s", linkFinalizer, link.Name)
			controllerutil.RemoveFinalizer(&link, linkFinalizer)
			if err := c.Update(ctx, &link); err != nil {
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
