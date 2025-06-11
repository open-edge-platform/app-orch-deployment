// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment_update

import (
	"net/http"

	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
)

func (s *TestSuite) TestUpdateDeploymentValidParams() {
	s.T().Parallel()
	testName := "TestUpdateDeploymentValidParams"
	// for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx, //app,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          testName,
	}
	deployID, code, err := deploymentutils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+deploymentutils.AppNginx+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")
	// }

	depl, code, err := deploymentutils.GetDeployment(s.AdmClient, deployID)
	s.Equal(http.StatusOK, code, "Expected HTTP status 200 for getting deployment details")
	s.NoError(err, "Failed to get deployment details")

	// depl.AppVersion = "v2.0.0" // Update the app version to a new valid version

	code, err = deploymentutils.UpdateDeployment(s.AdmClient, deployID, depl)
	s.Equal(http.StatusOK, code, "Expected HTTP status 200 for updating deployment")
	s.NoError(err, "Failed to update deployment with valid parameters")

	// for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
	// 	displayName := deploymentutils.GetDeployment(app, testName)
	// }
	// Update the deployment with valid parameters
	// originalDpConfigs := deploymentutils.CopyOriginalDpConfig(deploymentutils.DpConfigs)
	// defer func() { deploymentutils.DpConfigs = deploymentutils.CopyOriginalDpConfig(originalDpConfigs) }()
	// overrideValues := []map[string]any{
	// 	{
	// 		"appName":         deploymentutils.AppWordpress,
	// 		"targetNamespace": "",
	// 		"targetValues":    map[string]any{"service": map[string]any{"type": "NodePort"}},
	// 	},
	// 	{
	// 		"appName":         deploymentutils.AppNginx,
	// 		"targetNamespace": "",
	// 		"targetValues":    map[string]any{"service": map[string]any{"type": "NodePort"}},
	// 	},
	// }
	// err := deploymentutils.ResetThenChangeDpConfig(deploymentutils.AppWordpress, "overrideValues", overrideValues, originalDpConfigs)
	// s.NoError(err, "Failed to reset and change deployment configuration")
	// code, err := deploymentutils.UpdateDeployment(s.AdmClient, deploymentutils.AppWordpress, overrideValues[0])
	// s.Equal(http.StatusOK, code)
	// s.NoError(err, "Failed to update deployment with valid parameters")
	// code, err = deploymentutils.UpdateDeployment(s.AdmClient, deploymentutils.AppNginx, overrideValues[1])
	// s.Equal(http.StatusOK, code)
	// s.NoError(err, "Failed to update deployment with valid parameters")

	// Clean up the deployments created for this test
	// Note: Uncomment the following lines if you want to delete the deployments after the test
	// displayName := deploymentutils.FormDisplayName(deploymentutils.AppWordpress, testName)
	// err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
	// s.NoError(err)
	// displayName = deploymentutils.FormDisplayName(deploymentutils.AppNginx, testName)
	// err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
	// s.NoError(err)

	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}

func (s *TestSuite) TestUpdateDeploymentInvalidParams() {
	s.T().Parallel()
	testName := "TestUpdateDeploymentInvalidParams"
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppNginx, //app,
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
	s.Error(err, "Failed to update deployment with valid parameters")

	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}

// func (s *TestSuite) TestUpdateNonExistentDeployment() {
// 	s.T().Parallel()
// 	originalDpConfigs := deploymentutils.CopyOriginalDpConfig(deploymentutils.DpConfigs)
// 	defer func() { deploymentutils.DpConfigs = deploymentutils.CopyOriginalDpConfig(originalDpConfigs) }()

// 	overrideValues := []map[string]any{
// 		{
// 			"appName":         deploymentutils.AppWordpress,
// 			"targetNamespace": "",
// 			"targetValues":    map[string]any{"service": map[string]any{"type": "NodePort"}},
// 		},
// 	}
// 	testName := "TestUpdateNonExistentDeployment"
// 	err := deploymentutils.ResetThenChangeDpConfig(deploymentutils.AppWordpress, "overrideValues", overrideValues, originalDpConfigs)
// 	s.NoError(err, "Failed to reset and change deployment configuration")
// 	deploymentReq := deploymentutils.StartDeploymentRequest{
// 		AdmClient:         s.AdmClient,
// 		DpPackageName:     deploymentutils.AppWordpress,
// 		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
// 		DeploymentTimeout: deploymentutils.DeploymentTimeout,
// 		DeleteTimeout:     deploymentutils.DeleteTimeout,
// 		TestName:          testName,
// 	}

// 	_, code, err := deploymentutils.StartDeployment(deploymentReq)
// 	s.Equal(http.StatusOK, code)
// 	s.NoError(err, "Failed to create '"+deploymentutils.AppWordpress+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")

// 	displayName := deploymentutils.FormDisplayName(deploymentutils.AppWordpress, testName)
// 	err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
// 	s.NoError(err)
// }
