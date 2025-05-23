// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// NetworkServiceLister helps list NetworkServices.
// All objects returned here must be treated as read-only.
type NetworkServiceLister interface {
	// List lists all NetworkServices in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.NetworkService, err error)
	// Get retrieves the NetworkService from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.NetworkService, error)
	NetworkServiceListerExpansion
}

// networkServiceLister implements the NetworkServiceLister interface.
type networkServiceLister struct {
	indexer cache.Indexer
}

// NewNetworkServiceLister returns a new NetworkServiceLister.
func NewNetworkServiceLister(indexer cache.Indexer) NetworkServiceLister {
	return &networkServiceLister{indexer: indexer}
}

// List lists all NetworkServices in the indexer.
func (s *networkServiceLister) List(selector labels.Selector) (ret []*v1alpha1.NetworkService, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.NetworkService))
	})
	return ret, err
}

// Get retrieves the NetworkService from the index for a given name.
func (s *networkServiceLister) Get(name string) (*v1alpha1.NetworkService, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("networkservice"), name)
	}
	return obj.(*v1alpha1.NetworkService), nil
}
