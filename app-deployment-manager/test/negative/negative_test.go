// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package negative

/* TODO: Fix these tests, which are succeeding when they should be failing.
   The issue is that the REST API is returning a 200 OK status with an error message in the response body,
   instead of returning a 400 Bad Request status. Possible problem with ResetThenChangeDpConfig() not
   properly setting the configuration.

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
		s.T().Logf("Running test case with configKey: %s\n", test.configKey)
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
		s.Equal(http.StatusBadRequest, retCode)
		s.Error(err)  TODO: this is not returning an error as expected, need to research and fix
		if err != nil {
			s.Contains(err.Error(), test.expectedErr)
			s.Empty(deployID)
		}

		if !s.T().Failed() {
			s.T().Logf("successfully handled %s when creating %s deployment\n", test.expectedErr, test.deployment)
		}
	}
}
*/
