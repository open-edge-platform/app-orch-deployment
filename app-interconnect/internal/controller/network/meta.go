// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"encoding/json"
	"fmt"
	admv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	networkv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	"hash/fnv"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	helmReleaseNameAnnotation = "meta.helm.sh/release-name"
)

const (
	networkExposeServiceAnnotation = "network.app.edge-orchestrator.intel.com/expose-service"
	networkExposePortAnnotation    = "network.app.edge-orchestrator.intel.com/expose-port"
	networkExposePortsAnnotation   = "network.app.edge-orchestrator.intel.com/expose-ports"
)

const (
	deploymentIDKey        = "app.edge-orchestrator.intel.com/deployment-id"
	deploymentNamespaceKey = "app.edge-orchestrator.intel.com/deployment-namespace"
	deploymentNameKey      = "app.edge-orchestrator.intel.com/deployment-name"
	clusterNamespaceKey    = "app.edge-orchestrator.intel.com/cluster-namespace"
	clusterNameKey         = "app.edge-orchestrator.intel.com/cluster-name"
	networkNameKey         = "app.edge-orchestrator.intel.com/network-name"
)

const (
	networkNameField         = "metadata.networkName"
	deploymentUIDField       = "metadata.deploymentUID"
	deploymentNamespaceField = "metadata.deploymentNamespace"
	deploymentNameField      = "metadata.deploymentName"
	clusterNamespaceField    = "metadata.clusterNamespace"
	clusterNameField         = "metadata.clusterName"
	sourceNameField          = "metadata.sourceName"
	targetNameField          = "metadata.targetName"
)

func setupFieldIndexers(mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &admv1beta1.Deployment{}, networkNameField,
		func(rawObj client.Object) []string {
			deployment := rawObj.(*admv1beta1.Deployment)
			return []string{deployment.Spec.NetworkRef.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &admv1beta1.Deployment{}, deploymentUIDField,
		func(rawObj client.Object) []string {
			deployment := rawObj.(*admv1beta1.Deployment)
			return []string{string(deployment.UID)}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &admv1beta1.DeploymentCluster{}, deploymentUIDField,
		func(rawObj client.Object) []string {
			deploymentCluster := rawObj.(*admv1beta1.DeploymentCluster)
			return []string{deploymentCluster.Spec.DeploymentID}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &admv1beta1.DeploymentCluster{}, deploymentNamespaceField,
		func(rawObj client.Object) []string {
			deploymentCluster := rawObj.(*admv1beta1.DeploymentCluster)
			deploymentNamespace := deploymentCluster.Labels[deploymentNamespaceKey]
			if deploymentNamespace != "" {
				return []string{deploymentNamespace}
			}
			return nil
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &admv1beta1.DeploymentCluster{}, deploymentNameField,
		func(rawObj client.Object) []string {
			deploymentCluster := rawObj.(*admv1beta1.DeploymentCluster)
			deploymentNamespace := deploymentCluster.Labels[deploymentNameKey]
			if deploymentNamespace != "" {
				return []string{deploymentNamespace}
			}
			return nil
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &admv1beta1.DeploymentCluster{}, clusterNamespaceField,
		func(rawObj client.Object) []string {
			deploymentCluster := rawObj.(*admv1beta1.DeploymentCluster)
			return []string{deploymentCluster.Spec.Namespace}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &admv1beta1.DeploymentCluster{}, clusterNameField,
		func(rawObj client.Object) []string {
			deploymentCluster := rawObj.(*admv1beta1.DeploymentCluster)
			return []string{deploymentCluster.Spec.ClusterID}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkv1alpha1.NetworkCluster{}, networkNameField,
		func(rawObj client.Object) []string {
			networkCluster := rawObj.(*networkv1alpha1.NetworkCluster)
			return []string{networkCluster.Spec.NetworkRef.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkv1alpha1.NetworkLink{}, networkNameField,
		func(rawObj client.Object) []string {
			networkLink := rawObj.(*networkv1alpha1.NetworkLink)
			return []string{networkLink.Spec.NetworkRef.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkv1alpha1.NetworkLink{}, sourceNameField,
		func(rawObj client.Object) []string {
			networkLink := rawObj.(*networkv1alpha1.NetworkLink)
			return []string{networkLink.Spec.SourceClusterRef.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkv1alpha1.NetworkLink{}, targetNameField,
		func(rawObj client.Object) []string {
			networkLink := rawObj.(*networkv1alpha1.NetworkLink)
			return []string{networkLink.Spec.TargetClusterRef.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkv1alpha1.NetworkService{}, networkNameField,
		func(rawObj client.Object) []string {
			networkService := rawObj.(*networkv1alpha1.NetworkService)
			return []string{networkService.Spec.NetworkRef.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkv1alpha1.NetworkService{}, clusterNameField,
		func(rawObj client.Object) []string {
			networkService := rawObj.(*networkv1alpha1.NetworkService)
			return []string{networkService.Spec.ClusterRef.Name}
		}); err != nil {
		return err
	}
	return nil
}

func updateLabel(object client.Object, key, value string) bool {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if labels[key] == value {
		return false
	}
	labels[key] = value
	object.SetLabels(labels)
	return true
}

func updateLabels(object client.Object, labels map[string]string) bool {
	updated := false
	for key, value := range labels {
		if updateLabel(object, key, value) {
			updated = true
		}
	}
	return updated
}

func hasOwnerRef(object client.Object, owner client.Object) bool {
	for _, ownerRef := range object.GetOwnerReferences() {
		if ownerRef.APIVersion == owner.GetObjectKind().GroupVersionKind().GroupVersion().Identifier() && ownerRef.Kind == owner.GetObjectKind().GroupVersionKind().Kind && ownerRef.Name == owner.GetName() {
			return true
		}
	}
	return false
}

func hash(keys ...any) (string, error) {
	hasher := fnv.New32a()
	bytes, err := json.Marshal(keys)
	if err != nil {
		return "", err
	}
	_, err = hasher.Write(bytes)
	if err != nil {
		return "", err
	}
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32())), nil
}
