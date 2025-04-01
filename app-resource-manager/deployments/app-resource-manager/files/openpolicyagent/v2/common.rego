# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package resourcev2

import future.keywords.in

hasReadAccess {
	projectRole := sprintf("%s_ao-rw", [input.metadata.activeprojectid[0]])
    some role in input.metadata["realm_access/roles"] # iteration
    [projectRole][_] == role
}

hasWriteAccess {
	projectRole := sprintf("%s_ao-rw", [input.metadata.activeprojectid[0]])
    some role in input.metadata["realm_access/roles"] # iteration
    [projectRole][_] == role
}

hasVMConsoleAccess {
    projectRole := sprintf("%s_ao-rw", [input.metadata.activeprojectid[0]])
	some role in input.metadata["realm_access/roles"] # iteration
	[projectRole][_] == role
}

hasReadAccess {
	projectRole := "ao-m2m-rw"
	some role in input.metadata["realm_access/roles"] # iteration
	[projectRole][_] == role
}

hasWriteAccess {
	projectRole := "ao-m2m-rw"
	some role in input.metadata["realm_access/roles"] # iteration
	[projectRole][_] == role
}

hasVMConsoleAccess {
	projectRole := "ao-m2m-rw"
	some role in input.metadata["realm_access/roles"] # iteration
	[projectRole][_] == role
}