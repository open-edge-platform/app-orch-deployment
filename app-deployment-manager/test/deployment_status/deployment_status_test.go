// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"fmt"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"net/http"
)

func (s *TestSuite) TestDeploymentStatusWithNoLabels() {
	s.T().Parallel()
	testName := "DeploymentStatusWithNoLabels"
	deployemntReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppWordpress,
		DeploymentType:    deploymentutils.DeploymentTypeAutoScaling,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          testName,
	}
	_, code, err := deploymentutils.StartDeployment(deployemntReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err)
	status, code, err := deploymentutils.GetDeploymentsStatus(s.AdmClient, nil)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.NotZero(status.Total)
	s.NotZero(status.Running)
	displayName := deploymentutils.FormDisplayName(deploymentutils.AppWordpress, testName)
	err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
	s.NoError(err)

}

func (s *TestSuite) TestDeploymentStatusWithLabels() {
	s.T().Parallel()
	var labelsList []string

	deploymentReq := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.AppWordpress,
		DeploymentType:    deploymentutils.DeploymentTypeAutoScaling,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "DepStatusWithLabels",
	}
	_, code, err := deploymentutils.StartDeployment(deploymentReq)
	s.Equal(http.StatusOK, code)
	s.NoError(err)
	useDP := deploymentutils.DpConfigs[deploymentReq.DpPackageName].(map[string]any)
	labels := useDP["labels"].(map[string]string)
	for key, value := range labels {
		labelsList = append(labelsList, fmt.Sprintf("%s=%s", key, value))
	}

	status, code, err := deploymentutils.GetDeploymentsStatus(s.AdmClient, &labelsList)
	s.NoError(err)
	s.Equal(http.StatusOK, code)
	s.NotZero(status.Total)
	s.NotZero(status.Running)
	displayName := deploymentutils.FormDisplayName(deploymentutils.AppWordpress, deploymentReq.TestName)
	err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
	s.NoError(err)
}
