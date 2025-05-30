// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

func (s *TestSuite) TestNegativeCreateDeployment() {
	originalDpConfigs := CopyOriginalDpConfig(utils.DpConfigs)
	defer func() { utils.DpConfigs = CopyOriginalDpConfig(originalDpConfigs) }()

	tests := []struct {
		configKey   string
		configValue any
		deployment  string
		expectedErr string
	}{
		{"clusterId", "", "targeted", "missing targetClusters.labels or targetClusters.clusterId in request"},
		{"appNames", []string{""}, "targeted", "missing targetClusters.appName in request"},
		{"labels", map[string]string{}, "auto-scaling", "missing targetClusters.labels or targetClusters.clusterId in request"},
		{"overrideValues", []map[string]any{{"appName": "nginx", "targetNamespace": "", "targetValues": nil}}, "auto-scaling", "missing overrideValues.targetNamespace or overrideValues.values in request"},
	}

	for _, test := range tests {
		err := ResetThenChangeDpConfig("nginx", test.configKey, test.configValue, originalDpConfigs)
		s.NoError(err, "failed to reset "+test.configKey+" in deployment config")

		deploymentReq := utils.StartDeploymentRequest{
			AdmClient:      s.AdmClient,
			DpPackageName:  "nginx",
			DeploymentType: test.deployment,
			RetryDelay:     utils.DeploymentTimeout,
			TestName:       "NegativeCreateDeployment",
		}
		deployID, retCode, err := utils.StartDeployment(deploymentReq)
		s.Equal(retCode, http.StatusBadRequest)
		s.Error(err)
		s.Contains(err.Error(), test.expectedErr)
		s.Empty(deployID)

		if !s.T().Failed() {
			s.T().Logf("successfully handled %s when creating %s deployment\n", test.expectedErr, test.deployment)
		}
	}
}
