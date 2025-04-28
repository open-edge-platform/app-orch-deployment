// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	_, retCode, err := utils.StartDeployment(s.AdmClient, "wordpress", "targeted", 10)
	s.Equal(retCode, 200)
	s.NoError(err, "Failed to create 'wordpress-targeted' deployment")

	deployID, retCode, err = utils.StartDeployment(s.AdmClient, "nginx", "targeted", 10)
	s.Equal(retCode, 200)
	s.NoError(err, "Failed to create 'nginx-targeted' deployment")

}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	_, retCode, err := utils.StartDeployment(s.AdmClient, "wordpress", "auto-scaling", 10)
	s.Equal(retCode, 200)
	s.NoError(err, "Failed to create 'wordpress-auto-scaling' deployment")

	deployID, retCode, err = utils.StartDeployment(s.AdmClient, "nginx", "auto-scaling", 10)
	s.Equal(retCode, 200)
	s.NoError(err, "Failed to create 'nginx-auto-scaling' deployment")
}

func (s *TestSuite) TestCreateDiffDataDeployment() {
	// Make a copy of the original deployment configurations
	// to restore them after the test
	originalDpConfigs := CopyOriginalDpConfig(utils.DpConfigs)

	defer func() {
		utils.DpConfigs = CopyOriginalDpConfig(originalDpConfigs)
	}()

	// test with overrideValues
	serviceTypeNodePort := map[string]any{"service": map[string]any{"type": "NodePort"}}
	ResetThenChangeDpConfig("wordpress", "overrideValues", []map[string]any{{"appName": "nginx", "targetNamespace": "", "targetValues": serviceTypeNodePort}}, originalDpConfigs)

	_, retCode, err := utils.StartDeployment(s.AdmClient, "wordpress", "targeted", 10)
	s.Equal(retCode, 200)
	s.NoError(err, "Failed to create 'wordpress-targeted' deployment")
}
