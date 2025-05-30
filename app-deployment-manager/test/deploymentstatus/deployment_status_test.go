// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deploymentstatus

import (
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

func (s *TestSuite) TestRetrieveDeploymentStatusWithNoLabels() {
	deployemntReq := utils.StartDeploymentRequest{
		AdmClient:      s.AdmClient,
		DpPackageName:  utils.AppWordpress,
		DeploymentType: utils.DeploymentTypeAutoScaling,
		RetryDelay:     utils.DeploymentTimeout,
		TestName:       "DeploymentStatusWithNoLabels",
	}
	_, code, err := utils.StartDeployment(deployemntReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+utils.AppWordpress+"-"+utils.DeploymentTypeAutoScaling+"' deployment")
	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, nil)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.NotZero(status.Total)
	s.NotZero(status.Running)

}

func (s *TestSuite) TestDeploymentStatusWithLabelsFilter() {
	var labelsList []string

	deploymentReq := utils.StartDeploymentRequest{
		AdmClient:      s.AdmClient,
		DpPackageName:  utils.AppWordpress,
		DeploymentType: utils.DeploymentTypeAutoScaling,
		RetryDelay:     utils.DeploymentTimeout,
		TestName:       "DepStatusWithLabels",
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
}

func (s *TestSuite) TestDeploymentStateCountsVerification() {
	var labelsList []string
	deploymentReq := utils.StartDeploymentRequest{
		AdmClient:      s.AdmClient,
		DpPackageName:  utils.AppWordpress,
		DeploymentType: utils.DeploymentTypeAutoScaling,
		RetryDelay:     utils.DeploymentTimeout,
		TestName:       "DepStateCountsVerification",
	}
	useDP := utils.DpConfigs[deploymentReq.DpPackageName].(map[string]any)
	labels := useDP["labels"].(map[string]string)
	for key, value := range labels {
		labelsList = append(labelsList, fmt.Sprintf("%s=%s", key, value))
	}
	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Zero(*status.Deploying)
	s.Zero(*status.Down)
	s.Zero(*status.Error)
	currentTotal := *status.Total
	currentRunning := *status.Running

	_, code, err = utils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err)

	status, code, err = utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Equal(currentRunning+1, *status.Running)
	s.Equal(currentTotal+1, *status.Total)
	s.Zero(*status.Deploying)
	s.Zero(*status.Down)
	s.Zero(*status.Error)
}
