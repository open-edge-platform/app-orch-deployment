// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	testName := "CreateTargetedDeployment"
	for _, app := range []string{utils.AppWordpress, utils.AppNginx} {
		deploymentReq := utils.StartDeploymentRequest{
			AdmClient:      s.AdmClient,
			DpPackageName:  app,
			DeploymentType: utils.DeploymentTypeTargeted,
			RetryDelay:     utils.DeploymentTimeout,
			TestName:       testName,
		}
		_, code, err := utils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeTargeted+"' deployment")
	}

	for _, app := range []string{utils.AppWordpress, utils.AppNginx} {
		displayName := utils.FormDisplayName(app, testName)
		err := utils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, utils.RetryCount, utils.DeploymentTimeout)
		s.NoError(err)
	}
}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	testName := "CreateAutoScaleDeployment"
	for _, app := range []string{utils.AppWordpress, utils.AppNginx} {
		deploymentReq := utils.StartDeploymentRequest{
			AdmClient:      s.AdmClient,
			DpPackageName:  app,
			DeploymentType: utils.DeploymentTypeAutoScaling,
			RetryDelay:     utils.DeploymentTimeout,
			TestName:       testName,
		}
		_, code, err := utils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeAutoScaling+"' deployment")
	}
	for _, app := range []string{utils.AppWordpress, utils.AppNginx} {
		displayName := utils.FormDisplayName(app, testName)
		err := utils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, utils.RetryCount, utils.DeleteTimeout)
		s.NoError(err)
	}
}

func (s *TestSuite) TestCreateDiffDataDeployment() {
	originalDpConfigs := CopyOriginalDpConfig(utils.DpConfigs)
	defer func() { utils.DpConfigs = CopyOriginalDpConfig(originalDpConfigs) }()

	overrideValues := []map[string]any{
		{
			"appName":         utils.AppWordpress,
			"targetNamespace": "",
			"targetValues":    map[string]any{"service": map[string]any{"type": "NodePort"}},
		},
	}
	testName := "CreateDiffDataDeployment"
	err := ResetThenChangeDpConfig(utils.AppWordpress, "overrideValues", overrideValues, originalDpConfigs)
	s.NoError(err, "Failed to reset and change deployment configuration")
	deploymentReq := utils.StartDeploymentRequest{
		AdmClient:      s.AdmClient,
		DpPackageName:  utils.AppWordpress,
		DeploymentType: utils.DeploymentTypeTargeted,
		RetryDelay:     utils.DeploymentTimeout,
		TestName:       testName,
	}

	_, code, err := utils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+utils.AppWordpress+"-"+utils.DeploymentTypeTargeted+"' deployment")

	displayName := utils.FormDisplayName(utils.AppWordpress, testName)
	err = utils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, utils.RetryCount, utils.DeleteTimeout)
	s.NoError(err)
}
