# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

import future.keywords.in

# ao-m2m-rw
test_list_deployment_clusters_read_write_role if {
	not ListDeploymentClustersRequest with input as {
		"request": {"deplId": "deployment-1"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"uma_authorization",
		]},
	}
}

# ao-m2m-rw
test_list_deployment_clusters_read_role if {
	ListDeploymentClustersRequest with input as {
		"request": {"deplId": "deployment-2"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
