# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

# list deployments with ao-m2m-rw
test_list_deployments_read_role {
	ListDeploymentsRequest with input as {
		"request": {"labels": "customer=test"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}

# list ui extensions with ao-m2m-rw
test_list_ui_extensions_read_role {
	ListUIExtensionsRequest with input as {
		"request": {"service_name": ["test-ui-ext"]},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
