// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	s.T().Skip()
	for _, app := range []string{utils.AppWordpress, utils.AppNginx} {
		deploymentReq := utils.StartDeploymentRequest{
			AdmClient:      s.AdmClient,
			DpPackageName:  app,
			DeploymentType: utils.DeploymentTypeTargeted,
			RetryDelay:     utils.DeploymentTimeout,
			TestName:       "CreateTargetedDeployment",
		}
		_, code, err := utils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeTargeted+"' deployment")
	}
}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	s.T().Skip()
	for _, app := range []string{utils.AppWordpress, utils.AppNginx} {
		deploymentReq := utils.StartDeploymentRequest{
			AdmClient:      s.AdmClient,
			DpPackageName:  app,
			DeploymentType: utils.DeploymentTypeAutoScaling,
			RetryDelay:     utils.DeploymentTimeout,
			TestName:       "CreateAutoScaleDeployment",
		}
		_, code, err := utils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeAutoScaling+"' deployment")
	}
}

func (s *TestSuite) TestCreateDiffDataDeployment() {
	s.T().Skip()
	originalDpConfigs := CopyOriginalDpConfig(utils.DpConfigs)
	defer func() { utils.DpConfigs = CopyOriginalDpConfig(originalDpConfigs) }()

	overrideValues := []map[string]any{
		{
			"appName":         utils.AppWordpress,
			"targetNamespace": "",
			"targetValues":    map[string]any{"service": map[string]any{"type": "NodePort"}},
		},
	}
	err := ResetThenChangeDpConfig(utils.AppWordpress, "overrideValues", overrideValues, originalDpConfigs)
	s.NoError(err, "Failed to reset and change deployment configuration")
	deploymentReq := utils.StartDeploymentRequest{
		AdmClient:      s.AdmClient,
		DpPackageName:  utils.AppWordpress,
		DeploymentType: utils.DeploymentTypeTargeted,
		RetryDelay:     utils.DeploymentTimeout,
		TestName:       "CreateDiffDataDeployment",
	}

	_, code, err := utils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+utils.AppWordpress+"-"+utils.DeploymentTypeTargeted+"' deployment")
}
