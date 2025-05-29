// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	for _, app := range []string{utils.AppWordpress, utils.AppNginx} {
		_, code, err := utils.StartDeployment(s.AdmClient, app, utils.DeploymentTypeTargeted, utils.DeploymentTimeout, "CreateTargetedDeployment")
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeTargeted+"' deployment")
	}
}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	for _, app := range []string{utils.AppWordpress, utils.AppNginx} {
		_, code, err := utils.StartDeployment(s.AdmClient, app, utils.DeploymentTypeAutoScaling, utils.DeploymentTimeout, "CreateAutoScaleDeployment")
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+utils.DeploymentTypeAutoScaling+"' deployment")
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
	err := ResetThenChangeDpConfig(utils.AppWordpress, "overrideValues", overrideValues, originalDpConfigs)
	s.NoError(err, "Failed to reset and change deployment configuration")

	_, code, err := utils.StartDeployment(s.AdmClient, utils.AppWordpress, utils.DeploymentTypeTargeted, utils.DeploymentTimeout, "CreateDiffDataDeployment")
	s.Equal(http.StatusOK, code)
	s.NoError(err, "Failed to create '"+utils.AppWordpress+"-"+utils.DeploymentTypeTargeted+"' deployment")
}
