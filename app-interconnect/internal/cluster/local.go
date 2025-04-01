// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package clusterclient

import (
	"context"
	"k8s.io/client-go/rest"
)

type LocalClientOption func(*LocalClientOptions)

type LocalClientOptions struct {
}

func NewLocalClient(opts ...LocalClientOption) (Client, error) {
	var options LocalClientOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &localClient{}, nil
}

type localClient struct{}

func (c *localClient) GetClusterConfig(_ context.Context, _ ClusterID, _ ProjectID) (*rest.Config, error) {
	return rest.InClusterConfig()
}
