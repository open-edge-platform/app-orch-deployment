# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

import future.keywords

test_post_app_service_proxy_write_role_allowed {
	allow with input as {
		"metadata": {"realm_access/roles": [
			"default-roles-master",
			"offline_access",
			"ao-m2m-rw",
			"uma_authorization",
		]},
	}
}
