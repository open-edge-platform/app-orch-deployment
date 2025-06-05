// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"net/http"
)

// TestListDeploymentsAuthProjectID tests the list deployments method with invalid project ID
func (s *TestSuite) TestListDeploymentsAuthProjectID() {
	s.T().Parallel()
	admClient, err := clients.CreateAdmClient(s.deploymentRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

	deployments, retCode, err := deploymentutils.DeploymentsList(admClient)
	s.Equal(retCode, http.StatusForbidden)
	s.Error(err)
	s.Empty(deployments)

	if !s.T().Failed() {
		s.T().Logf("successfully handled invalid projectid to list deployments\n")
	}
}

// TestGetDeploymentAuthProjectID tests the get deployment method with invalid project ID
func (s *TestSuite) TestGetDeploymentAuthProjectID() {
	s.T().Parallel()
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "GetDeploymentAuthProjectID",
	}
	deployID, retCode, err := deploymentutils.StartDeployment(deploymentReq)
	s.Equal(retCode, http.StatusOK)
	s.NoError(err)

	admClient, err := clients.CreateAdmClient(s.deploymentRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

	deployment, retCode, err := deploymentutils.GetDeployment(admClient, deployID)
	s.Equal(retCode, http.StatusForbidden)
	s.Error(err)
	s.Empty(deployment)

	if !s.T().Failed() {
		s.T().Logf("successfully handled invalid projectid to get deployment\n")
	}

	// Clean up the deployment created for this test
	displayName := deploymentutils.FormDisplayName(deploymentutils.AppNginx, deploymentReq.TestName)
	err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
	s.NoError(err)
}
