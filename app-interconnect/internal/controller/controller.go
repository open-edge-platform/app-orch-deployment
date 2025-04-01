// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func New(mgr manager.Manager, clusters clusterclient.Client) *Controller {
	return &Controller{
		Client:   mgr.GetClient(),
		Cache:    mgr.GetCache(),
		Scheme:   mgr.GetScheme(),
		Config:   mgr.GetConfig(),
		Events:   mgr.GetEventRecorderFor("interconnect-manager"),
		Clusters: clusters,
	}
}

type ManagedController interface {
	Setup(mgr manager.Manager) error
}

type Controller struct {
	client.Client
	Cache    cache.Cache
	Scheme   *runtime.Scheme
	Config   *rest.Config
	Events   record.EventRecorder
	Clusters clusterclient.Client
}
