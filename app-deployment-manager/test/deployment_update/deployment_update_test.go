// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment_update

import (
	"net/http"

	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
)

func (s *TestSuite) TestUpdateDeploymentValidParams() {
	testName := "TestUpdateDeplValidParams"
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppWordpress,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          testName,
	}
	deployID, code, err := deploymentutils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+deploymentutils.AppWordpress+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")

	deployment, code, err := deploymentutils.GetDeployment(s.AdmClient, deployID)
	s.Equal(http.StatusOK, code, "Expected HTTP status 200 for getting deployment details")
	s.NoError(err, "Failed to get deployment details")

	s.T().Logf("deployment: %+v", deployment)

	// TODO: some more modification the deployment object to update?
	deployment.AppVersion = "0.1.1"

	code, err = deploymentutils.UpdateDeployment(s.AdmClient, deployID, deployment)
	s.Equal(http.StatusOK, code, "Expected HTTP status 200 for updating deployment")
	s.NoError(err, "Failed to update deployment with valid parameters")

	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}

func (s *TestSuite) TestUpdateDeploymentInvalidParams() {
	// // s.T().Parallel() // Disabled to run tests sequentially
	testName := "TestUpdateDeplInvalidParams"
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          testName,
	}
	deployID, code, err := deploymentutils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+deploymentutils.AppNginx+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")

	depl, code, err := deploymentutils.GetDeployment(s.AdmClient, deployID)
	s.Equal(http.StatusOK, code, "Expected HTTP status 200 for getting deployment details")
	s.NoError(err, "Failed to get deployment details")

	depl.TargetClusters = nil
	depl.AllAppTargetClusters = nil

	code, err = deploymentutils.UpdateDeployment(s.AdmClient, deployID, depl)
	s.Equal(http.StatusBadRequest, code, "Expected HTTP status 400 for updating deployment")
	s.Error(err, "Should have failed to update deployment with invalid parameters")

	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}

func (s *TestSuite) TestUpdateNonExistentDeployment() {
	testName := "TestUpdateNonExistentDepl"
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          testName,
	}
	deployID, code, err := deploymentutils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+deploymentutils.AppNginx+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")

	depl, code, err := deploymentutils.GetDeployment(s.AdmClient, deployID)
	s.Equal(http.StatusOK, code, "Expected HTTP status 200 for getting deployment details")
	s.NoError(err, "Failed to get deployment details")

	deployID = "ae25322c-8309-4ed6-81a1-a6ecb77bbc64" // Non-existent deployment ID
	code, err = deploymentutils.UpdateDeployment(s.AdmClient, deployID, depl)
	s.Equal(http.StatusNotFound, code, "Expected HTTP status 404 for updating non-existent deployment")
	s.Error(err, "Should have failed to update deployment with valid parameters")

	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}
