// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	deploymentv1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"

	"net/http"
)

func (s *TestSuite) TestDeleteExistingDeployment() {
	s.T().Parallel()
	// Create a deployment to be deleted
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "DeleteExistingDeployment",
	}
	deploymentID, status, err := deploymentutils.StartDeployment(deploymentReq)
	s.NoError(err)
	s.Equal(http.StatusOK, status, "Expected HTTP status 200 for successful deployment creation")

	// Delete the created deployment
	status, err = deploymentutils.DeleteDeployment(s.AdmClient, deploymentID)
	s.NoError(err)
	s.Equal(http.StatusOK, status, "Expected HTTP status 200 for successful deletion")

	s.T().Logf("successfully deleted deployment with ID: %s", deploymentID)
}

func (s *TestSuite) TestDeleteNonExistentDeployment() {
	s.T().Parallel()
	// Attempt to delete a deployment that does not exist
	deploymentID := "non-existent-deployment"
	status, err := deploymentutils.DeleteDeployment(s.AdmClient, deploymentID)
	s.T().Log(err)
	s.Equal(http.StatusNotFound, status, "Expected HTTP status 404 for non-existent deployment deletion")
	s.T().Logf("successfully handled deletion of non-existent deployment with ID: %s", deploymentID)
}

func (s *TestSuite) TestDeleteDeploymentParentOnlyDeleteType() {
	s.T().Parallel()
	// Create a deployment to be deleted
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "DelDeploymentParentOnly",
	}
	deploymentID, status, err := deploymentutils.StartDeployment(deploymentReq)
	s.NoError(err)
	s.Equal(http.StatusOK, status, "Expected HTTP status 200 for successful deployment creation")

	// Delete the created deployment with delete type
	status, err = deploymentutils.DeleteDeploymentWithDeleteType(s.AdmClient, deploymentID, deploymentv1.DeleteType_PARENT_ONLY)
	s.NoError(err)
	s.Equal(http.StatusOK, status, "Expected HTTP status 200 for successful deletion with delete type")

	s.T().Logf("successfully deleted deployment with ID: %s using delete type", deploymentID)
}

func (s *TestSuite) TestDeleteDeploymentAllDeleteType() {
	s.T().Parallel()
	// Create a deployment to be deleted
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "DelDeploymentAll",
	}
	deploymentID, status, err := deploymentutils.StartDeployment(deploymentReq)
	s.NoError(err)
	s.Equal(http.StatusOK, status, "Expected HTTP status 200 for successful deployment creation")

	// Delete the created deployment with delete type
	status, err = deploymentutils.DeleteDeploymentWithDeleteType(s.AdmClient, deploymentID, deploymentv1.DeleteType_ALL)
	s.NoError(err)
	s.Equal(http.StatusOK, status, "Expected HTTP status 200 for successful deletion with delete type")

	s.T().Logf("successfully deleted deployment with ID: %s using delete type", deploymentID)
}
