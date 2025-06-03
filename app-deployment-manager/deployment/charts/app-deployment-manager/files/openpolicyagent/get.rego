# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

import future.keywords.in

GetDeploymentRequest if {
	hasReadAccess
}

GetDeploymentsStatusRequest if {
	hasReadAccess
}

GetClusterRequest if {
	hasReadAccess
}
