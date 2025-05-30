// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

// TestListDeploymentsAuthProjectID tests the list deployments method with invalid project ID
func (s *TestSuite) TestListDeploymentsAuthProjectID() {
	admClient, err := utils.CreateClient(deploymentRESTServerUrl, token, "invalidprojectid")
	s.NoError(err)

	deployments, retCode, err := DeploymentsList(admClient)
	s.Equal(retCode, http.StatusForbidden)
	s.Error(err)
	s.Empty(deployments)

	if !s.T().Failed() {
		s.T().Logf("successfully handled invalid projectid to list deployments\n")
	}
}

// TestGetDeploymentAuthProjectID tests the get deployment method with invalid project ID
func (s *TestSuite) TestGetDeploymentAuthProjectID() {

	deploymentReq := utils.StartDeploymentRequest{
		AdmClient:      s.AdmClient,
		DpPackageName:  utils.AppNginx,
		DeploymentType: utils.DeploymentTypeTargeted,
		RetryDelay:     utils.DeploymentTimeout,
		TestName:       "GetDeploymentAuthProjectID",
	}
	deployID, retCode, err := utils.StartDeployment(deploymentReq)
	s.Equal(retCode, http.StatusForbidden)
	s.NoError(err)
	
	admClient, err := utils.CreateClient(deploymentRESTServerUrl, token, "invalidprojectid")
	s.NoError(err)

	deployment, retCode, err := utils.GetDeployment(admClient, deployID)
	s.Equal(retCode, http.StatusForbidden)
	s.Error(err)
	s.Empty(deployment)

	if !s.T().Failed() {
		s.T().Logf("successfully handled invalid projectid to get deployment\n")
	}
}
