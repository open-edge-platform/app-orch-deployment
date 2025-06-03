# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

import future.keywords

test_post_app_service_proxy_write_role_allowed if {
	allow with input as {
		"metadata": {
		    "activeprojectid": ["5d0cef5c-9981-4987-a67e-3e207783218b"],
		    "realm_access/roles": [
                "default-roles-master",
                "offline_access",
                "5d0cef5c-9981-4987-a67e-3e207783218b_ao-rw",
                "uma_authorization",
    		]
        },
	}
}

test_post_app_service_proxy_write_role_m2m_allowed if {
	allow with input as {
		"metadata": {
		    "activeprojectid": ["5d0cef5c-9981-4987-a67e-3e207783218b"],
		    "realm_access/roles": [
                "default-roles-master",
                "offline_access",
                "ao-m2m-rw",
                "uma_authorization",
    		]
        },
	}
}

test_post_app_service_proxy_read_role_denied if {
	not allow with input as {
		"metadata": {
		    "activeprojectid": ["5d0cef5c-9981-4987-a67e-3e207783218b"],
		    "realm_access/roles": [
                "default-roles-master",
                "offline_access",
                "5d0cef5c-9981-4987-a67e-3e207783218b_ao-r",
                "uma_authorization",
    		]
        },
	}
}
