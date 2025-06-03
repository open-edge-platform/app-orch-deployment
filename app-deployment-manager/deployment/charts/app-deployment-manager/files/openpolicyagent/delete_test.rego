# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

# delete deployment with app-deployment-manager-write role
test_delete_deployment_write_role if {
	DeleteDeploymentRequest with input as {
		"request": {"depl_id": "5d0cef5c-9981-4987-a67e-3e207783218b"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
