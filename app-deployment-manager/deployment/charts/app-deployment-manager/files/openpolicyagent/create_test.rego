# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

# create a deployment with ao-m2m-rw
test_create_deployment_write_role if {
	CreateDeploymentRequest with input as {
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
		},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}

# create a deployment with ao-m2m-rw
test_create_deployment_read_role if {
	not CreateDeploymentRequest with input as {
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
		},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"uma_authorization",
		]},
	}
}
