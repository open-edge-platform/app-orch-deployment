# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

test_has_write_access if {
	hasWriteAccess with input as {
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

test_has_read_access if {
	hasReadAccess with input as {
		"request": {"depl_id": "5d0cef5c-9981-4987-a67e-3e207783218b"},
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
