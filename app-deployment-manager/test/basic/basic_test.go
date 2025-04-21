// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
)

// TestBasics tests the basic functionality of the deployment manager
func (s *TestSuite) TestBasics() {
	s.T().Parallel()
	res, err := s.AdmClient.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	s.NoError(err)
	s.Equal(200, res.StatusCode())
}
