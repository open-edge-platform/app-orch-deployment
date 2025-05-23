// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeNetworkClusters implements NetworkClusterInterface
type FakeNetworkClusters struct {
	Fake *FakeNetworkV1alpha1
}

var networkclustersResource = v1alpha1.SchemeGroupVersion.WithResource("networkclusters")

var networkclustersKind = v1alpha1.SchemeGroupVersion.WithKind("NetworkCluster")

// Get takes name of the networkCluster, and returns the corresponding networkCluster object, and an error if there is any.
func (c *FakeNetworkClusters) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NetworkCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(networkclustersResource, name), &v1alpha1.NetworkCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NetworkCluster), err
}

// List takes label and field selectors, and returns the list of NetworkClusters that match those selectors.
func (c *FakeNetworkClusters) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NetworkClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(networkclustersResource, networkclustersKind, opts), &v1alpha1.NetworkClusterList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.NetworkClusterList{ListMeta: obj.(*v1alpha1.NetworkClusterList).ListMeta}
	for _, item := range obj.(*v1alpha1.NetworkClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested networkClusters.
func (c *FakeNetworkClusters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(networkclustersResource, opts))
}

// Create takes the representation of a networkCluster and creates it.  Returns the server's representation of the networkCluster, and an error, if there is any.
func (c *FakeNetworkClusters) Create(ctx context.Context, networkCluster *v1alpha1.NetworkCluster, opts v1.CreateOptions) (result *v1alpha1.NetworkCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(networkclustersResource, networkCluster), &v1alpha1.NetworkCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NetworkCluster), err
}

// Update takes the representation of a networkCluster and updates it. Returns the server's representation of the networkCluster, and an error, if there is any.
func (c *FakeNetworkClusters) Update(ctx context.Context, networkCluster *v1alpha1.NetworkCluster, opts v1.UpdateOptions) (result *v1alpha1.NetworkCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(networkclustersResource, networkCluster), &v1alpha1.NetworkCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NetworkCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeNetworkClusters) UpdateStatus(ctx context.Context, networkCluster *v1alpha1.NetworkCluster, opts v1.UpdateOptions) (*v1alpha1.NetworkCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(networkclustersResource, "status", networkCluster), &v1alpha1.NetworkCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NetworkCluster), err
}

// Delete takes name of the networkCluster and deletes it. Returns an error if one occurs.
func (c *FakeNetworkClusters) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(networkclustersResource, name, opts), &v1alpha1.NetworkCluster{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNetworkClusters) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(networkclustersResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.NetworkClusterList{})
	return err
}

// Patch applies the patch and returns the patched networkCluster.
func (c *FakeNetworkClusters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NetworkCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(networkclustersResource, name, pt, data, subresources...), &v1alpha1.NetworkCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NetworkCluster), err
}
