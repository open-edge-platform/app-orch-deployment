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
	_, code, err := utils.StartDeployment(s.AdmClient, utils.AppWordpress, utils.DeploymentTypeAutoScaling, utils.DeploymentTimeout, "DeploymentStatusWithNoLabels")
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
		_, code, err := utils.StartDeployment(s.AdmClient, app, utils.DeploymentTypeAutoScaling, utils.DeploymentTimeout, "DeploymentStatusWithLabels")
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeAutoScaling+"' deployment")
		useDP := utils.DpConfigs[app].(map[string]any)
		if labels, ok := useDP["labelsList"].([]string); ok {
			labelsList = append(labelsList, labels...)
		}
	}

	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Equal(int32(1), *status.Running)
	s.Equal(int32(1), *status.Total)
}

func (s *TestSuite) TestDeploymentStateCountsVerification() {
	var labelsList []string
	var deploymentIDs []string
	for _, app := range []string{utils.AppWordpress} {
		deployID, code, err := utils.StartDeployment(s.AdmClient, app, utils.DeploymentTypeAutoScaling, utils.DeploymentTimeout, "TestDeploymentStateCountsVerification")
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeAutoScaling+"' deployment")
		useDP := utils.DpConfigs[app].(map[string]any)
		if labels, ok := useDP["labelsList"].([]string); ok {
			labelsList = append(labelsList, labels...)
		}
		deploymentIDs = append(deploymentIDs, deployID)
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
