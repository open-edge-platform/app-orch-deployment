# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

# get kubeConfig with ao-m2m-rw
#
# ao-m2m-rw
test_get_kube_config_read_role if {
	GetKubeConfigRequest with input as {
		"request": {"cluster_id": "cluster-46f4a3485e28"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
