// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

// MaxItems validation constants for API requests and responses
const (
	// Maximum number of labels allowed in cluster-related requests
	MAX_LABELS_PER_REQUEST_CLUSTERS = 100

	// Maximum number of labels allowed in deployment-related requests
	MAX_LABELS_PER_REQUEST_DEPLOYMENTS = 20

	// Maximum number of clusters returned in a single response
	MAX_CLUSTERS_RESPONSE = 1000

	// Maximum number of deployments returned in a single response
	MAX_DEPLOYMENTS_RESPONSE = 1000
)
