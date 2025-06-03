# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package deploymentv1

import future.keywords.in

ListClustersRequest if {
	hasReadAccess
}
