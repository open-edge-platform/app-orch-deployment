// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package ingresshandler

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/config"
)

type IngressKind string

const (
	nginx   IngressKind = "nginx"
	traefik IngressKind = "traefik"
)

type Option func(*IngressHandler)

type IngressHandler interface {
	// Init performs handler specific initialization.
	// Panic for non-nil errors.
	Init()

	// GetRoute retrieves an ingress route obj from the Kubernetes Cluster.
	// obj must be a struct pointer so that obj can be updated with the response
	// returned by the Server.
	GetRoute(ctx context.Context, key client.ObjectKey, opts ...client.GetOption) (client.Object, error)

	// CreateRoute creates an ingress route obj for the given proxy endpoint
	// to the Kubernetes Cluster.
	CreateRoute(ctx context.Context, apiExtension *v1beta1.APIExtension, endpoint v1beta1.ProxyEndpoint) error
}

// New returns IngressHandler.
func New(scheme *runtime.Scheme, client client.Client) (IngressHandler, error) {
	cfg := config.GetAPIExtensionConfig()

	switch cfg.IngressKind {
	case string(nginx):
		return &nginxIngressHandler{
			scheme: scheme,
			client: client,
		}, nil
	case string(traefik):
		return &traefikIngressHandler{
			scheme: scheme,
			client: client,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported ingress kind %s", cfg.IngressKind)
	}
}
