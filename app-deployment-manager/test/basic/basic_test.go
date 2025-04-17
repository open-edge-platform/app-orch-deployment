// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
)

// TestRest tests basics of exercising the REST API of the catalog service.
func (s *TestSuite) TestBasics() {
	s.T().Parallel()
	res, err := s.client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	s.NoError(err)
	s.Equal(200, res.StatusCode())
	s.TearDownTest(context.TODO())
}
