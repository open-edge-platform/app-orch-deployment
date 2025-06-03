// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"net/http"
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	testName := "CreateTargetedDeployment"
	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		deploymentReq := deploymentutils.StartDeploymentRequest{
			AdmClient:         s.AdmClient,
			DpPackageName:     app,
			DeploymentType:    deploymentutils.DeploymentTypeTargeted,
			DeploymentTimeout: deploymentutils.DeploymentTimeout,
			DeleteTimeout:     deploymentutils.DeleteTimeout,
			TestName:          testName,
		}
		_, code, err := deploymentutils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")
	}

	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	testName := "CreateAutoScaleDeployment"
	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		deploymentReq := deploymentutils.StartDeploymentRequest{
			AdmClient:         s.AdmClient,
			DpPackageName:     app,
			DeploymentType:    deploymentutils.DeploymentTypeAutoScaling,
			DeploymentTimeout: deploymentutils.DeploymentTimeout,
			DeleteTimeout:     deploymentutils.DeleteTimeout,
			TestName:          testName,
		}
		_, code, err := deploymentutils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+deploymentutils.DeploymentTypeAutoScaling+"' deployment")
	}
	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}

func (s *TestSuite) TestCreateDiffDataDeployment() {
	originalDpConfigs := deploymentutils.CopyOriginalDpConfig(deploymentutils.DpConfigs)
	defer func() { deploymentutils.DpConfigs = deploymentutils.CopyOriginalDpConfig(originalDpConfigs) }()

	overrideValues := []map[string]any{
		{
			"appName":         deploymentutils.AppWordpress,
			"targetNamespace": "",
			"targetValues":    map[string]any{"service": map[string]any{"type": "NodePort"}},
		},
	}
	testName := "CreateDiffDataDeployment"
	err := deploymentutils.ResetThenChangeDpConfig(deploymentutils.AppWordpress, "overrideValues", overrideValues, originalDpConfigs)
	s.NoError(err, "Failed to reset and change deployment configuration")
	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppWordpress,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          testName,
	}

	_, code, err := deploymentutils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+deploymentutils.AppWordpress+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")

	displayName := deploymentutils.FormDisplayName(deploymentutils.AppWordpress, testName)
	err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
	s.NoError(err)
}
