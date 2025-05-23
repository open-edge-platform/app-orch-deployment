// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/interconnect/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeServices implements ServiceInterface
type FakeServices struct {
	Fake *FakeInterconnectV1alpha1
}

var servicesResource = v1alpha1.SchemeGroupVersion.WithResource("services")

var servicesKind = v1alpha1.SchemeGroupVersion.WithKind("Service")

// Get takes name of the service, and returns the corresponding service object, and an error if there is any.
func (c *FakeServices) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Service, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(servicesResource, name), &v1alpha1.Service{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Service), err
}

// List takes label and field selectors, and returns the list of Services that match those selectors.
func (c *FakeServices) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ServiceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(servicesResource, servicesKind, opts), &v1alpha1.ServiceList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ServiceList{ListMeta: obj.(*v1alpha1.ServiceList).ListMeta}
	for _, item := range obj.(*v1alpha1.ServiceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested services.
func (c *FakeServices) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(servicesResource, opts))
}

// Create takes the representation of a service and creates it.  Returns the server's representation of the service, and an error, if there is any.
func (c *FakeServices) Create(ctx context.Context, service *v1alpha1.Service, opts v1.CreateOptions) (result *v1alpha1.Service, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(servicesResource, service), &v1alpha1.Service{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Service), err
}

// Update takes the representation of a service and updates it. Returns the server's representation of the service, and an error, if there is any.
func (c *FakeServices) Update(ctx context.Context, service *v1alpha1.Service, opts v1.UpdateOptions) (result *v1alpha1.Service, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(servicesResource, service), &v1alpha1.Service{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Service), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeServices) UpdateStatus(ctx context.Context, service *v1alpha1.Service, opts v1.UpdateOptions) (*v1alpha1.Service, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(servicesResource, "status", service), &v1alpha1.Service{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Service), err
}

// Delete takes name of the service and deletes it. Returns an error if one occurs.
func (c *FakeServices) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(servicesResource, name, opts), &v1alpha1.Service{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeServices) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(servicesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ServiceList{})
	return err
}

// Patch applies the patch and returns the patched service.
func (c *FakeServices) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Service, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(servicesResource, name, pt, data, subresources...), &v1alpha1.Service{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Service), err
}
