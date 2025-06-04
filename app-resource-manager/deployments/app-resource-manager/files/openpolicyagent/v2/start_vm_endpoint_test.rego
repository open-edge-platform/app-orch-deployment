# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package resourcev2

test_not_allow_start_vm_read_role if {
	not StartVirtualMachineRequest with input as {
		"request": {
			"app_id": "testapp",
			"cluster_id": "testcluester",
			"virtual_machine_id": "5d0cef5c-9981-4987-a67e-3e207783218b",
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

test_start_vm_write_role if {
	StartVirtualMachineRequest with input as {
		"request": {
			"app_id": "testapp",
			"cluster_id": "testcluester",
			"virtual_machine_id": "5d0cef5c-9981-4987-a67e-3e207783218b",
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
