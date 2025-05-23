// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	networkv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	versioned "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/clientset/versioned"
	internalinterfaces "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/listers/network/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// NetworkLinkInformer provides access to a shared informer and lister for
// NetworkLinks.
type NetworkLinkInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.NetworkLinkLister
}

type networkLinkInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewNetworkLinkInformer constructs a new informer for NetworkLink type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewNetworkLinkInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredNetworkLinkInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredNetworkLinkInformer constructs a new informer for NetworkLink type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredNetworkLinkInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NetworkV1alpha1().NetworkLinks().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NetworkV1alpha1().NetworkLinks().Watch(context.TODO(), options)
			},
		},
		&networkv1alpha1.NetworkLink{},
		resyncPeriod,
		indexers,
	)
}

func (f *networkLinkInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNetworkLinkInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *networkLinkInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&networkv1alpha1.NetworkLink{}, f.defaultInformer)
}

func (f *networkLinkInformer) Lister() v1alpha1.NetworkLinkLister {
	return v1alpha1.NewNetworkLinkLister(f.Informer().GetIndexer())
}
