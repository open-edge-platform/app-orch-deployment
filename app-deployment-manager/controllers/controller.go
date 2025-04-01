// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func New(name string, mgr manager.Manager) *Controller {
	return &Controller{
		Client: mgr.GetClient(),
		Cache:  mgr.GetCache(),
		Scheme: mgr.GetScheme(),
		Config: mgr.GetConfig(),
		Events: mgr.GetEventRecorderFor(name),
	}
}

type ManagedController interface {
	SetupWithManager(mgr manager.Manager) error
}

type Controller struct {
	client.Client
	Cache  cache.Cache
	Scheme *runtime.Scheme
	Config *rest.Config
	Events record.EventRecorder
}
