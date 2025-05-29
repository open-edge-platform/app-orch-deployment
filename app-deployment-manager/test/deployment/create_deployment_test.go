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
	for _, app := range []string{AppWordpress, AppNginx} {
		_, code, err := utils.StartDeployment(s.AdmClient, app, DeploymentTypeTargeted, DeploymentTimeout)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+DeploymentTypeTargeted+"' deployment")
	}
}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	for _, app := range []string{AppWordpress, AppNginx} {
		_, code, err := utils.StartDeployment(s.AdmClient, app, DeploymentTypeAutoScaling, DeploymentTimeout)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+DeploymentTypeAutoScaling+"' deployment")
	}
}

func (s *TestSuite) TestCreateDiffDataDeployment() {
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

func (s *TestSuite) TestRetrieveDeploymentStatus() {
	for _, app := range []string{AppWordpress, AppNginx} {
		_, code, err := utils.StartDeployment(s.AdmClient, app, DeploymentTypeTargeted, DeploymentTimeout)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+DeploymentTypeTargeted+"' deployment")

		status, code, err := utils.GetDeploymentsStatus(s.AdmClient, &[]string{})
		s.NoError(err)
		s.Equal(http.StatusOK, code)
		s.Equal(2, status.Running)
		s.Equal(2, status.Total)

	}
}
