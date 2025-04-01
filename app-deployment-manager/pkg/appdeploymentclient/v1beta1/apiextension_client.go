// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	apiExtensionsv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

// APIExtensionInterface has methods to work with apiExtensionClient resources.
type APIExtensionInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*apiExtensionsv1beta1.APIExtensionList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*apiExtensionsv1beta1.APIExtension, error)
	Create(ctx context.Context, apiExtension *apiExtensionsv1beta1.APIExtension, opts metav1.CreateOptions) (*apiExtensionsv1beta1.APIExtension, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

// apiExtensionClient implements APIExtensionInterface
type apiExtensionClient struct {
	appDeploymentClient rest.Interface
	ns                  string
}

func (c *apiExtensionClient) List(ctx context.Context, opts metav1.ListOptions) (*apiExtensionsv1beta1.APIExtensionList, error) {
	result := apiExtensionsv1beta1.APIExtensionList{}
	err := c.appDeploymentClient.
		Get().
		Namespace(c.ns).
		Resource(apiExtensionsv1beta1.APIExtensionResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *apiExtensionClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*apiExtensionsv1beta1.APIExtension, error) {
	result := apiExtensionsv1beta1.APIExtension{}
	err := c.appDeploymentClient.
		Get().
		Namespace(c.ns).
		Resource(apiExtensionsv1beta1.APIExtensionResource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *apiExtensionClient) Create(ctx context.Context, apiExtension *apiExtensionsv1beta1.APIExtension, opts metav1.CreateOptions) (*apiExtensionsv1beta1.APIExtension, error) {
	result := apiExtensionsv1beta1.APIExtension{}
	err := c.appDeploymentClient.
		Post().
		Namespace(c.ns).
		Resource(apiExtensionsv1beta1.APIExtensionResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(apiExtension).
		Do(ctx).
		Into(&result)

	return &result, err
}

// Delete deletes based on name.
func (c *apiExtensionClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.appDeploymentClient.Delete().
		Namespace(c.ns).
		Resource(apiExtensionsv1beta1.APIExtensionResource).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}
