// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	grafanav1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

// GrafanaExtensionInterface has methods to work with grafanaExtensionClient resources.
type GrafanaExtensionInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*grafanav1beta1.GrafanaExtensionList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*grafanav1beta1.GrafanaExtension, error)
	Create(ctx context.Context, grafanaExtension *grafanav1beta1.GrafanaExtension, opts metav1.CreateOptions) (*grafanav1beta1.GrafanaExtension, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

// grafanaExtensionClient implements GrafanaExtensionInterface
type grafanaExtensionClient struct {
	appDeploymentClient rest.Interface
	ns                  string
}

func (c *grafanaExtensionClient) List(ctx context.Context, opts metav1.ListOptions) (*grafanav1beta1.GrafanaExtensionList, error) {
	result := grafanav1beta1.GrafanaExtensionList{}
	err := c.appDeploymentClient.
		Get().
		Namespace(c.ns).
		Resource(grafanav1beta1.GrafanaExtensionResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *grafanaExtensionClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*grafanav1beta1.GrafanaExtension, error) {
	result := grafanav1beta1.GrafanaExtension{}
	err := c.appDeploymentClient.
		Get().
		Namespace(c.ns).
		Resource(grafanav1beta1.GrafanaExtensionResource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *grafanaExtensionClient) Create(ctx context.Context, grafanaExtension *grafanav1beta1.GrafanaExtension, opts metav1.CreateOptions) (*grafanav1beta1.GrafanaExtension, error) {
	result := grafanav1beta1.GrafanaExtension{}
	err := c.appDeploymentClient.
		Post().
		Namespace(c.ns).
		Resource(grafanav1beta1.GrafanaExtensionResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(grafanaExtension).
		Do(ctx).
		Into(&result)

	return &result, err
}

// Delete deletes based on name.
func (c *grafanaExtensionClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.appDeploymentClient.Delete().
		Namespace(c.ns).
		Resource(grafanav1beta1.GrafanaExtensionResource).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}
