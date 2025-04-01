// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package clusterclient

import (
	"context"
	"k8s.io/client-go/rest"
)

type ProjectID string

type ClusterID string

type Client interface {
	GetClusterConfig(ctx context.Context, clusterID ClusterID, projectID ProjectID) (*rest.Config, error)
}
