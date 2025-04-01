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

// DeploymentInterface has methods to work with deploymentClient resources.
type DeploymentInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*deploymentsv1beta1.DeploymentList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentsv1beta1.Deployment, error)
	Create(ctx context.Context, deployment *deploymentsv1beta1.Deployment, opts metav1.CreateOptions) (*deploymentsv1beta1.Deployment, error)
	Update(ctx context.Context, name string, deployment *deploymentsv1beta1.Deployment, opts metav1.UpdateOptions) (*deploymentsv1beta1.Deployment, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

// deploymentClient implements DeploymentInterface
type deploymentClient struct {
	apporchClient rest.Interface
	ns            string
}

func (c *deploymentClient) List(ctx context.Context, opts metav1.ListOptions) (*deploymentsv1beta1.DeploymentList, error) {
	result := deploymentsv1beta1.DeploymentList{}
	err := c.apporchClient.
		Get().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.DeploymentsResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *deploymentClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentsv1beta1.Deployment, error) {
	result := deploymentsv1beta1.Deployment{}
	err := c.apporchClient.
		Get().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.DeploymentsResource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *deploymentClient) Create(ctx context.Context, deployment *deploymentsv1beta1.Deployment, opts metav1.CreateOptions) (*deploymentsv1beta1.Deployment, error) {
	result := deploymentsv1beta1.Deployment{}
	err := c.apporchClient.
		Post().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.DeploymentsResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(deployment).
		Do(ctx).
		Into(&result)

	return &result, err
}

// Delete deletes Deployment based on name.
func (c *deploymentClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.apporchClient.Delete().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.DeploymentsResource).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// Update updates deployment CR object. Returns the updated deployment CR object.
func (c *deploymentClient) Update(ctx context.Context, name string, deployment *deploymentsv1beta1.Deployment, opts metav1.UpdateOptions) (result *deploymentsv1beta1.Deployment, err error) {
	result = &deploymentsv1beta1.Deployment{}
	err = c.apporchClient.Put().
		Namespace(c.ns).
		Resource(deploymentsv1beta1.DeploymentsResource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(deployment).
		Do(ctx).
		Into(result)
	return
}
