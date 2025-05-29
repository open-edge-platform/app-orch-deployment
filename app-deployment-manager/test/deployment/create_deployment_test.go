// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

const (
	// DeploymentTypeTargeted represents the targeted deployment type
	DeploymentTypeTargeted = "targeted"
	// DeploymentTypeAutoScaling represents the auto-scaling deployment type
	DeploymentTypeAutoScaling = "auto-scaling"

	// AppWordpress represents the WordPress application name
	AppWordpress = "wordpress"
	// AppNginx represents the Nginx application name
	AppNginx = "nginx"

	// DeploymentTimeout represents the timeout in seconds for deployment operations
	DeploymentTimeout = 10
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	s.T().Skip()
	for _, app := range []string{AppWordpress, AppNginx} {
		_, code, err := utils.StartDeployment(s.AdmClient, app, DeploymentTypeTargeted, DeploymentTimeout)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+DeploymentTypeTargeted+"' deployment")
	}
}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	s.T().Skip()
	for _, app := range []string{AppWordpress, AppNginx} {
		_, code, err := utils.StartDeployment(s.AdmClient, app, DeploymentTypeAutoScaling, DeploymentTimeout)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+DeploymentTypeAutoScaling+"' deployment")
	}
}

func (s *TestSuite) TestCreateDiffDataDeployment() {
	s.T().Skip()
	originalDpConfigs := CopyOriginalDpConfig(utils.DpConfigs)
	defer func() { utils.DpConfigs = CopyOriginalDpConfig(originalDpConfigs) }()

	overrideValues := []map[string]any{
		{
			"appName":         AppNginx,
			"targetNamespace": "",
			"targetValues":    map[string]any{"service": map[string]any{"type": "NodePort"}},
		},
	}
	err := ResetThenChangeDpConfig(AppWordpress, "overrideValues", overrideValues, originalDpConfigs)
	s.NoError(err, "Failed to reset and change deployment configuration")

	_, code, err := utils.StartDeployment(s.AdmClient, AppWordpress, DeploymentTypeTargeted, DeploymentTimeout)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+AppWordpress+"-"+DeploymentTypeTargeted+"' deployment")
}
func (s *TestSuite) TestRetrieveDeploymentStatusWithNoLabels() {
	s.T().Skip()
	_, code, err := utils.StartDeployment(s.AdmClient, AppWordpress, DeploymentTypeAutoScaling, DeploymentTimeout)
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+AppWordpress+"-"+DeploymentTypeAutoScaling+"' deployment")
	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, nil)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.NotZero(status.Total)
	s.NotZero(status.Running)

}

func (s *TestSuite) TestDeploymentStatusWithLabelsFilter() {
	s.T().Skip()
	var labelsList []string
	for _, app := range []string{AppWordpress, AppNginx} {
		_, code, err := utils.StartDeployment(s.AdmClient, app, DeploymentTypeAutoScaling, DeploymentTimeout)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+DeploymentTypeAutoScaling+"' deployment")
		useDP := utils.DpConfigs[app].(map[string]any)
		if labels, ok := useDP["labelsList"].([]string); ok {
			labelsList = append(labelsList, labels...)
		}
	}

	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Equal(int32(2), *status.Running)
	s.Equal(int32(2), *status.Total)
}

func (s *TestSuite) TestDeploymentStateCountsVerification() {
	var labelsList []string
	var deploymentIDs []string
	for _, app := range []string{AppWordpress, AppNginx} {
		deployID, code, err := utils.StartDeployment(s.AdmClient, app, DeploymentTypeAutoScaling, DeploymentTimeout)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+DeploymentTypeAutoScaling+"' deployment")
		useDP := utils.DpConfigs[app].(map[string]any)
		if labels, ok := useDP["labelsList"].([]string); ok {
			labelsList = append(labelsList, labels...)
		}
		deploymentIDs = append(deploymentIDs, deployID)
	}

	status, code, err := utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Equal(int32(2), *status.Running)
	s.Equal(int32(2), *status.Total)
	s.Zero(*status.Deploying)
	s.Zero(*status.Down)
	s.Zero(*status.Error)

	err = utils.DeleteDeployment(s.AdmClient, deploymentIDs[0])
	s.NoError(err, "Failed to delete deployment "+deploymentIDs[0])
	deployment, retCode, err := utils.GetDeployment(s.AdmClient, deploymentIDs[0])
	s.Equal(retCode, http.StatusOK)
	s.NoError(err)
	s.Empty(deployment)

	status, code, err = utils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.Equal(int32(1), *status.Running)
	s.Equal(int32(1), *status.Total)
	s.Zero(*status.Deploying)
	s.Zero(*status.Down)
	s.Zero(*status.Error)
}
