// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"net/http"
)

// TestListDeploymentsInvalidProjectID tests the list deployments method with invalid project ID
func (s *TestSuite) TestListDeploymentsInvalidProjectID() {
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

// TestGetDeploymentInvalidProjectID tests the get deployment method with invalid project ID
func (s *TestSuite) TestGetDeploymentInvalidProjectID() {
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
func (s *TestSuite) TestDeleteDeploymentInvalidProjectID() {
	s.T().Parallel()
	// Create a deployment to be deleted
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "DeleteDeploymentAuthProjectID",
	}
	deploymentID, status, err := deploymentutils.StartDeployment(deploymentReq)
	s.NoError(err)
	s.Equal(http.StatusOK, status, "Expected HTTP status 200 for successful deployment creation")

	admClient, err := clients.CreateAdmClient(s.deploymentRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

	status, err = deploymentutils.DeleteDeployment(admClient, deploymentID)
	s.Equal(http.StatusForbidden, status)
	s.Error(err)

	// Clean up the deployment created for this test
	displayName := deploymentutils.FormDisplayName(deploymentutils.AppNginx, deploymentReq.TestName)
	err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
	s.NoError(err)

	s.T().Logf("successfully handled invalid projectid to delete deployment\n")

}
