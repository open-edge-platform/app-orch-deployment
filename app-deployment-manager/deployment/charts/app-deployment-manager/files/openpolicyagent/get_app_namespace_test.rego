# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

# get app namespace with ao-m2m-rw
#
# ao-m2m-rw
#
test_get_app_namespace_read_role if {
	GetAppNamespaceRequest with input as {
		"request": {"app_id": "b-bf3059c9-a156-5a24-841c-37957ec6d185"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
