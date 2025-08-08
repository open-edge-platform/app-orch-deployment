// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

// MaxItems validation constants for API requests and responses
const (
	// Maximum number of labels allowed in cluster-related requests
	MaxLabelsPerRequestClusters = 100

	// Maximum number of labels allowed in deployment-related requests
	MaxLabelsPerRequestDeployments = 20

	// Maximum number of clusters returned in a single response
	MaxClustersResponse = 1000

	// Maximum number of deployments returned in a single response
	MaxDeploymentsResponse = 1000
)
