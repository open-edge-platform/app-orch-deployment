// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
)

func (s *TestSuite) TestNegativeCreateDeployment() {
	// Make a copy of the original deployment configurations
	// to restore them after the test
	originalDpConfigs := CopyOriginalDpConfig(utils.DpConfigs)

	defer func() {
		utils.DpConfigs = CopyOriginalDpConfig(originalDpConfigs)
	}()

	err := ResetThenChangeDpConfig("nginx", "clusterId", "", originalDpConfigs)
	s.NoError(err, "failed to reset clusterId in deployment config")

	deployID, retCode, err := utils.StartDeployment(s.AdmClient, "nginx", "targeted", 10)
	s.Equal(retCode, 400)
	s.Error(err)
	s.Contains(err.Error(), "missing targetClusters.labels or targetClusters.clusterId in request")
	s.Empty(deployID)
	if !s.T().Failed() {
		s.T().Logf("successfully handled missing targetClusters.clusterId when creating targeted deployment\n")
	}

	err = ResetThenChangeDpConfig("nginx", "appNames", []string{""}, originalDpConfigs)
	s.NoError(err, "failed to reset appNames in deployment config")

	deployID, retCode, err = utils.StartDeployment(s.AdmClient, "nginx", "targeted", 10)
	s.Equal(retCode, 400)
	s.Error(err)
	s.Contains(err.Error(), "missing targetClusters.appName in request")
	s.Empty(deployID)

	if !s.T().Failed() {
		s.T().Logf("successfully handled missing targetClusters.appName when creating deployment\n")
	}

	err = ResetThenChangeDpConfig("nginx", "labels", map[string]string{}, originalDpConfigs)
	s.NoError(err, "failed to reset labels in deployment config")

	deployID, retCode, err = utils.StartDeployment(s.AdmClient, "nginx", "auto-scaling", 10)
	s.Equal(retCode, 400)
	s.Error(err)
	s.Contains(err.Error(), "missing targetClusters.labels or targetClusters.clusterId in request")
	s.Empty(deployID)

	if !s.T().Failed() {
		s.T().Logf("successfully handled missing targetClusters.labels when creating auto-scaling deployment\n")
	}

	err = ResetThenChangeDpConfig("nginx", "overrideValues", []map[string]any{{"appName": "nginx", "targetNamespace": "", "targetValues": nil}}, originalDpConfigs)
	s.NoError(err, "failed to reset overrideValues in deployment config")

	deployID, retCode, err = utils.StartDeployment(s.AdmClient, "nginx", "auto-scaling", 10)
	s.Equal(retCode, 400)
	s.Error(err)
	s.Contains(err.Error(), "missing overrideValues.targetNamespace or overrideValues.values in request")
	s.Empty(deployID)

	if !s.T().Failed() {
		s.T().Logf("successfully handled missing overrideValues.targetNamespace or overrideValues.values when creating deployment\n")
	}

	// this test will eventually fail due to app name not being found
	// need to fix in ADM, update s.Contains with error output and then uncomment this test

	// updateDeploymentConfig("nginx", "appNames", []string{"invalid"}, originalDpConfigs)

	// deployID, retCode, err = utils.StartDeployment(s.AdmClient, "nginx", "auto-scaling", 10)
	// s.Equal(retCode, 200)
	// s.Error(err)
	// s.Contains(err.Error(), "app not found due to invalid appname")
	// s.Empty(deployID)

	// if !s.T().Failed() {
	// 	s.T().Logf("successfully handled app not found due to invalid appname\n")
	// }
}
