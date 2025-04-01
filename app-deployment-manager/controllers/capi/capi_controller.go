// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package capi

import (
	"context"
	"fmt"
	v1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/controllers"
	"github.com/open-edge-platform/orch-library/go/dazl"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

var log = dazl.GetPackageLogger()

func AddClusterController(mgr manager.Manager) error {
	ctrl := &ClusterController{
		Controller: controllers.New("capi", mgr),
	}

	return ctrl.SetupWithManager(mgr)
}

type ClusterController struct {
	*controllers.Controller
}

func (r *ClusterController) SetupWithManager(mgr manager.Manager) (err error) {
	log.Info("Setting up K8s Cluster API (CAPI) Controller")
	return ctrl.NewControllerManagedBy(mgr).
		Named("capi").
		For(&capiv1beta1.Cluster{}).
		Owns(&fleetv1alpha1.Cluster{}).
		Complete(r)

}

func (r *ClusterController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling Request %s", request.NamespacedName)
	capiCluster := &capiv1beta1.Cluster{}
	err := r.Get(ctx, request.NamespacedName, capiCluster)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}
	log.Infof("Reconciling CAPI Resource %s, Phase %s", request.NamespacedName, capiCluster.Status.GetTypedPhase())
	switch capiCluster.Status.GetTypedPhase() {
	case capiv1beta1.ClusterPhasePending:
		// No op
	case capiv1beta1.ClusterPhaseProvisioning:
		// No op
	case capiv1beta1.ClusterPhaseProvisioned:
		return r.reconcilePhaseProvisioned(ctx, capiCluster)
		// Reconcile Cluster Phase Provisioned
	case capiv1beta1.ClusterPhaseDeleting:
		// No op for now
	case capiv1beta1.ClusterPhaseFailed:
		// No op for now
	}

	return reconcile.Result{}, nil
}

func (r *ClusterController) reconcilePhaseProvisioned(ctx context.Context, capiCluster *capiv1beta1.Cluster) (reconcile.Result, error) {
	// Wait until capi cluster is fully ready
	if !(capiCluster.Status.ControlPlaneReady) || !(capiCluster.Status.InfrastructureReady) {
		log.Infof("CapiCluster %s is not ready, requeuing", capiCluster.Name)
		return reconcile.Result{RequeueAfter: time.Second * 30}, nil
	}

	// Get kubeconfig secret
	kubeConfigSecret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{
		Name:      fmt.Sprintf("%s-kubeconfig", capiCluster.Name),
		Namespace: capiCluster.Namespace,
	}, kubeConfigSecret)
	if err != nil {
		log.Warnf(fmt.Sprintf("Failed to get secret %s-kubeconfig", capiCluster.Name))
		return reconcile.Result{}, err
	}

	fleetCluster := &fleetv1alpha1.Cluster{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      capiCluster.Name,
		Namespace: capiCluster.Namespace,
	}, fleetCluster)

	if err != nil {
		if errors.IsNotFound(err) {
			// Fleet cluster resource not found, create a new one
			log.Infof("Creating FleetCluster %s ", capiCluster.Name)
			fleetCluster := &fleetv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        capiCluster.Name,
					Namespace:   capiCluster.Namespace,
					Labels:      capiCluster.Labels,
					Annotations: capiCluster.Annotations,
				},
				Spec: fleetv1alpha1.ClusterSpec{
					KubeConfigSecret: kubeConfigSecret.Name,
				},
			}
			// Set the owner reference to the CAPI cluster
			if err := controllerutil.SetControllerReference(capiCluster, fleetCluster, r.Scheme); err != nil {
				return reconcile.Result{}, err
			}

			// TODO : Add this label in the capi resource when it is created to keep all labels in sync.
			log.Infof("Infrastructure cluster Name %s", capiCluster.Spec.InfrastructureRef.Name)
			fleetCluster.ObjectMeta.Labels[string(v1beta1.CapiInfraName)] = capiCluster.Spec.InfrastructureRef.Name

			err = r.Create(ctx, fleetCluster)
			if err != nil {
				return reconcile.Result{}, err
			}
			log.Infof("Created FleetCluster %s", capiCluster.Name)
			return reconcile.Result{}, nil
		}
		log.Warnf(fmt.Sprintf("Unable to get FleetCluster %s", capiCluster.Name))
	} else {
		// Update the FleetCluster resource
		log.Infof("Updating FleetCluster %s", fleetCluster.Name)
		fleetCluster.ObjectMeta.Labels = capiCluster.Labels
		fleetCluster.ObjectMeta.Annotations = capiCluster.Annotations
		fleetCluster.ObjectMeta.Labels[string(v1beta1.CapiInfraName)] = capiCluster.Spec.InfrastructureRef.Name
		err = r.Update(ctx, fleetCluster)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Warnf(fmt.Sprintf("Unable to update FleetCluster %s since resource not found", fleetCluster.Name))
				return reconcile.Result{}, nil
			}
			log.Warnf(fmt.Sprintf("Unable to update FleetCluster %s", fleetCluster.Name))
		}
	}

	return reconcile.Result{}, nil
}
