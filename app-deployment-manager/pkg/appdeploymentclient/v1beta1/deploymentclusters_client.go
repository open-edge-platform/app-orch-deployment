// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	deploymentsv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

// DeploymentClusterInterface has methods to work with deploymentClusterClient resources.
type DeploymentClusterInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*deploymentsv1beta1.DeploymentClusterList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentsv1beta1.DeploymentCluster, error)
}

// deploymentClusterClient implements DeploymentClusterInterface
type deploymentClusterClient struct {
	crClient rest.Interface
	ns       string
}

func (c *deploymentClusterClient) List(ctx context.Context, opts metav1.ListOptions) (*deploymentsv1beta1.DeploymentClusterList, error) {
	result := deploymentsv1beta1.DeploymentClusterList{}
	err := c.crClient.
		Get().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.DeploymentClustersResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *deploymentClusterClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentsv1beta1.DeploymentCluster, error) {
	result := deploymentsv1beta1.DeploymentCluster{}
	err := c.crClient.
		Get().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.DeploymentClustersResource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}
