# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package resourcev2

test_delete_pod_write_role if {
	DeletePodRequest with input as {
		"request": {
			"app_id": "testapp",
			"cluster_id": "testcluester",
			"pod_name": "testpod",
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

test_not_allow_delete_pod_read_role if {
	not DeletePodRequest with input as {
		"request": {
			"app_id": "testapp",
			"cluster_id": "testcluester",
			"pod_name": "testpod",
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
