# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

# ao-m2m-rw
test_get_deployment_write_role {
	not GetDeploymentRequest with input as {
		"request": {"depl_id": "5d0cef5c-9981-4987-a67e-3e207783218b"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"uma_authorization",
		]},
	}
}

# ao-m2m-rw
test_get_deployment_read_role {
	GetDeploymentRequest with input as {
		"request": {"depl_id": "5d0cef5c-9981-4987-a67e-3e207783218b"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}

# get deployments status with ao-m2m-rw
test_get_deployments_status_write_role {
	not GetDeploymentsStatusRequest with input as {
		"request": {"labels": "customer=test"},
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

# get deployments status with app-deploymenet-manager-read-role
test_get_deployments_status_read_role {
	GetDeploymentsStatusRequest with input as {
		"request": {"labels": "customer=test"},
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

# get api extension with app-deploymenet-manager-read-role
test_get_api_extension_read_role {
	GetAPIExtensionRequest with input as {
		"request": {"name": "test-apiext"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}

# ao-m2m-rw
test_get_cluster_read_role {
	GetClusterRequest with input as {
		"request": {"cluster_id": "cluster-01234567"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
