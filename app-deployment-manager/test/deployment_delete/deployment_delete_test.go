// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"net/http"
)

func (s *TestSuite) TestDeleteNonExistentDeployment() {
	s.T().Parallel()
	// Attempt to delete a deployment that does not exist
	deploymentID := "non-existent-deployment"
	status, err := deploymentutils.DeleteDeployment(s.AdmClient, deploymentID)
	s.T().Log(err)
	s.Equal(http.StatusNotFound, status, "Expected HTTP status 404 for non-existent deployment deletion")
	s.T().Logf("successfully handled deletion of non-existent deployment with ID: %s", deploymentID)
}
