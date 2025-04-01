# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

import future.keywords

allow if {
	projectRole := sprintf("%s_ao-rw", [input.metadata.activeprojectid[0]])
    some role in input.metadata["realm_access/roles"] # iteration
    [projectRole][_] == role
}

allow if {
	projectRole := "ao-m2m-rw"
	some role in input.metadata["realm_access/roles"] # iteration
	[projectRole][_] == role
}

