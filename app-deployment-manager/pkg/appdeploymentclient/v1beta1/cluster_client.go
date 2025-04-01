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

// ClusterInterface has methods to work with deploymentClusterClient resources.
type ClusterInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*deploymentsv1beta1.ClusterList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentsv1beta1.Cluster, error)
}

// clusterClient implements ClusterInterface
type clusterClient struct {
	crClient rest.Interface
	ns       string
}

func (c *clusterClient) List(ctx context.Context, opts metav1.ListOptions) (*deploymentsv1beta1.ClusterList, error) {
	result := deploymentsv1beta1.ClusterList{}
	err := c.crClient.
		Get().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.ClustersResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *clusterClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentsv1beta1.Cluster, error) {
	result := deploymentsv1beta1.Cluster{}
	err := c.crClient.
		Get().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.ClustersResource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}
