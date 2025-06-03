// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"net/http"
)

func (s *TestSuite) TestNegativeCreateDeployment() {
	originalDpConfigs := deploymentutils.CopyOriginalDpConfig(deploymentutils.DpConfigs)
	defer func() { deploymentutils.DpConfigs = deploymentutils.CopyOriginalDpConfig(originalDpConfigs) }()

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
		err := deploymentutils.ResetThenChangeDpConfig("nginx", test.configKey, test.configValue, originalDpConfigs)
		s.NoError(err, "failed to reset "+test.configKey+" in deployment config")

		deploymentReq := deploymentutils.StartDeploymentRequest{
			AdmClient:         s.AdmClient,
			DpPackageName:     "nginx",
			DeploymentType:    test.deployment,
			DeploymentTimeout: deploymentutils.DeploymentTimeout,
			DeleteTimeout:     deploymentutils.DeleteTimeout,
			TestName:          "NegativeCreateDeployment",
		}
		deployID, retCode, err := deploymentutils.StartDeployment(deploymentReq)
		s.Equal(retCode, http.StatusBadRequest)
		s.Error(err)
		s.Contains(err.Error(), test.expectedErr)
		s.Empty(deployID)

		if !s.T().Failed() {
			s.T().Logf("successfully handled %s when creating %s deployment\n", test.expectedErr, test.deployment)
		}
	}
}
