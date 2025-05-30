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

	for _, app := range []string{utils.AppWordpress} {
		deploymentReq := utils.StartDeploymentRequest{
			AdmClient:      s.AdmClient,
			DpPackageName:  app,
			DeploymentType: utils.DeploymentTypeAutoScaling,
			RetryDelay:     utils.DeploymentTimeout,
			TestName:       "DepStatusWithLabels",
		}
		_, code, err := utils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeAutoScaling+"' deployment")
	}
	for key, value := range utils.DpConfigs["labels"].(map[string]string) {
		labelsList = append(labelsList, fmt.Sprintf("%s=%s", key, value))
	}

	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Equal(int32(1), *status.Running)
	s.Equal(int32(1), *status.Total)
}

func (s *TestSuite) TestDeploymentStateCountsVerification() {
	var deploymentIDs []string
	var labelsList []string

	for _, app := range []string{utils.AppWordpress} {
		deploymentReq := utils.StartDeploymentRequest{
			AdmClient:      s.AdmClient,
			DpPackageName:  app,
			DeploymentType: utils.DeploymentTypeAutoScaling,
			RetryDelay:     utils.DeploymentTimeout,
			TestName:       "DepStateCountsVerification",
		}
		deployID, code, err := utils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeAutoScaling+"' deployment")

		deploymentIDs = append(deploymentIDs, deployID)
	}
	for key, value := range utils.DpConfigs["labels"].(map[string]string) {
		labelsList = append(labelsList, fmt.Sprintf("%s=%s", key, value))
	}

	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Equal(int32(1), *status.Running)
	s.Equal(int32(1), *status.Total)
	s.Zero(*status.Deploying)
	s.Zero(*status.Down)
	s.Zero(*status.Error)
	displayName := fmt.Sprintf("%s-%s", utils.AppWordpress, utils.DeploymentTypeAutoScaling)
	err = utils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, 20, 10)
	s.NoError(err)
	deployment, retCode, err := utils.GetDeployment(s.AdmClient, deploymentIDs[0])
	s.Equal(retCode, http.StatusNotFound)
	s.Empty(deployment)

	status, code, err = utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Equal(int32(0), *status.Running)
	s.Equal(int32(0), *status.Total)
	s.Zero(*status.Deploying)
	s.Zero(*status.Down)
	s.Zero(*status.Error)
}
