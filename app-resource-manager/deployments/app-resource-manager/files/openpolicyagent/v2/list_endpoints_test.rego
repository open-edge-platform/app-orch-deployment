# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package resourcev2

test_allow_list_endpoints_read_role {
	ListAppEndpointsRequest with input as {
		"request": {
			"app_id": "testapp",
			"cluster_id": "testcluester",
		},
		"metadata": {
			"client": ["catalog-cli"],
			"realm_access/roles": [
				"default-roles-master",
				"offline_access",
				"ao-m2m-rw",
				"uma_authorization",
			],
		},
	}
}

test_not_allow_list_endpoints_write_role {
	not ListAppEndpointsRequest with input as {
		"request": {
			"app_id": "testapp",
			"cluster_id": "testcluester",
		},
		"metadata": {
			"client": ["catalog-cli"],
			"realm_access/roles": [
				"default-roles-master",
				"offline_access",
				"uma_authorization",
			],
		},
	}
}
