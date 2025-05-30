// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

func (s *TestSuite) TestDeploymentStatusWithNoLabels() {
	testName := "DeploymentStatusWithNoLabels"
	deployemntReq := utils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     utils.AppWordpress,
		DeploymentType:    utils.DeploymentTypeAutoScaling,
		DeploymentTimeout: utils.DeploymentTimeout,
		DeleteTimeout:     utils.DeleteTimeout,
		TestName:          testName,
	}
	_, code, err := utils.StartDeployment(deployemntReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err)
	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, nil)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.NotZero(status.Total)
	s.NotZero(status.Running)
	displayName := utils.FormDisplayName(utils.AppWordpress, testName)
	err = utils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, utils.RetryCount, utils.DeleteTimeout)
	s.NoError(err)

}

func (s *TestSuite) TestDeploymentStatusWithLabels() {
	var labelsList []string

	deploymentReq := utils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     utils.AppWordpress,
		DeploymentType:    utils.DeploymentTypeAutoScaling,
		DeploymentTimeout: utils.DeploymentTimeout,
		DeleteTimeout:     utils.DeleteTimeout,
		TestName:          "DepStatusWithLabels",
	}
	_, code, err := utils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err)
	useDP := utils.DpConfigs[deploymentReq.DpPackageName].(map[string]any)
	labels := useDP["labels"].(map[string]string)
	for key, value := range labels {
		labelsList = append(labelsList, fmt.Sprintf("%s=%s", key, value))
	}

	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.NotZero(status.Total)
	s.NotZero(status.Running)
	displayName := utils.FormDisplayName(utils.AppWordpress, deploymentReq.TestName)
	err = utils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, utils.RetryCount, utils.DeleteTimeout)
	s.NoError(err)
}
