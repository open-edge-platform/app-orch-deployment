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

	// Maximum page size for list requests
	MaxPageSize = 100

	// Maximum number of clusters returned in a single response
	MaxClustersResponse = 1000

	// Maximum number of deployments returned in a single response
	MaxDeploymentsResponse = 1000

	// Pattern validation regex for labels
	LabelPattern = `(^$)|^[a-z0-9]([-_.=,a-z0-9]{0,198}[a-z0-9])?$`

	// Pattern validation regex for deployment and cluster IDs
	IDPattern = `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$`

	// Pattern validation regex for app version
	AppVersionPattern = `^[a-z0-9][a-z0-9-.]{0,18}[a-z0-9]{0,1}$`
)
