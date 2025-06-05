# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

#  list clusters with ao-m2m-rw
test_list_clusters_write_role if {
	not ListClustersRequest with input as {
		"request": {"labels": "customer=test"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"uma_authorization",
		]},
	}
}

# list clusters with ao-m2m-rw
test_list_clusters_read_role if {
	ListClustersRequest with input as {
		"request": {"labels": "customer=test"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
