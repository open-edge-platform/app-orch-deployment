// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"net/http"
)

var listDeploymentsMethods = map[string]int{
	http.MethodGet:    200,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPut:    405,
	http.MethodPost:   400, // 400 is returned when the request body is empty since the post method is used to create deployment
}

var listDeploymentsPerClusterMethods = map[string]int{
	http.MethodGet:    200,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPut:    405,
	http.MethodPost:   405,
}

var getDeleteDeploymentMethods = map[string]int{
	http.MethodGet:    200,
	http.MethodDelete: 200,
	http.MethodPatch:  405,
	http.MethodPut:    400,
	http.MethodPost:   405,
}

var getDeploymentsStatusMethods = map[string]int{
	http.MethodGet:    200,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPut:    405,
	http.MethodPost:   405,
}

var listDeploymentClustersMethods = map[string]int{
	http.MethodGet:    200,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPut:    405,
	http.MethodPost:   405,
}

// TestListDeploymentsMethod tests the list deployments method
func (s *TestSuite) TestListDeploymentsMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments", deploymentRESTServerUrl)
	for method, expectedStatus := range listDeploymentsMethods {
		res, err := utils.CallMethod(url, method, token, projectID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)
		s.T().Logf("list deployments method: %s (%d)\n", method, res.StatusCode)
	}
}

// TestListDeploymentsPerClusterMethod tests the list deployments per cluster method
func (s *TestSuite) TestListDeploymentsPerClusterMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/clusters/%s", deploymentRESTServerUrl, deploy.TestClusterID)
	for method, expectedStatus := range listDeploymentsPerClusterMethods {
		res, err := utils.CallMethod(url, method, token, projectID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)
		s.T().Logf("list deployments per cluster method: %s (%d)\n", method, res.StatusCode)
	}
}

// TestGetDeleteDeploymentMethod tests the get and delete deployment method
func (s *TestSuite) TestGetDeleteDeploymentMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s", deploymentRESTServerUrl, deployID)
	for method, expectedStatus := range getDeleteDeploymentMethods {
		res, err := utils.CallMethod(url, method, token, projectID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)
		if method == http.MethodDelete {
			_, err = deploy.CreateDeployment(admclient, dpConfigName, dpDisplayName, 10)
			s.NoError(err)

			deployID = deploy.FindDeploymentIDByDisplayName(admclient, dpDisplayName)
			s.NoError(err)
		}

		s.T().Logf("get deployment method: %s (%d)\n", method, res.StatusCode)
	}
}

// TestGetDeploymentsStatusMethod tests the get deployments status method
func (s *TestSuite) TestGetDeploymentsStatusMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/summary/deployments_status", deploymentRESTServerUrl)
	for method, expectedStatus := range getDeploymentsStatusMethods {
		res, err := utils.CallMethod(url, method, token, projectID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)
		s.T().Logf("get deployments status method: %s (%d)\n", method, res.StatusCode)
	}
}

// TestListDeploymentClustersMethod tests the list deployment clusters method
func (s *TestSuite) TestListDeploymentClustersMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s/clusters", deploymentRESTServerUrl, deployID)
	for method, expectedStatus := range listDeploymentClustersMethods {
		res, err := utils.CallMethod(url, method, token, projectID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)
		s.T().Logf("list deployment clusters method: %s (%d)\n", method, res.StatusCode)
	}
}
