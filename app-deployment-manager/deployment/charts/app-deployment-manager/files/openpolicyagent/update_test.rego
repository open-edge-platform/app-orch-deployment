# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

# update deployment with ao-m2m-rw
test_update_deployment_write_role if {
	UpdateDeploymentRequest with input as {
		"request": {
			"displayName": "test display name",
			"publisherName": "intel",
			"profileName": "testing",
			"appVersion": "0.1.1",
			"appName": "test-name",
			"overrideValues": [{
				"appName": "test-wordpress",
				"targetNamespace": "test-targetnamespace",
				"values": {"service": {"type": "test-type"}},
			}],
			"targetClusters": [{
				"appName": "wordpress",
				"labels": {"color": "red"},
			}],
			"depl_id": "5d0cef5c-9981-4987-a67e-3e207783218b",
		},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}

# update deployment with ao-m2m-rw
test_update_deployment_read_role if {
	not UpdateDeploymentRequest with input as {
		"request": {
			"displayName": "test display name",
			"publisherName": "intel",
			"profileName": "testing",
			"appVersion": "0.1.1",
			"appName": "test-name",
			"overrideValues": [{
				"appName": "test-wordpress",
				"targetNamespace": "test-targetnamespace",
				"values": {"service": {"type": "test-type"}},
			}],
			"targetClusters": [{
				"appName": "wordpress",
				"labels": {"color": "red"},
			}],
			"depl_id": "5d0cef5c-9981-4987-a67e-3e207783218b",
		},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"uma_authorization",
		]},
	}
}
