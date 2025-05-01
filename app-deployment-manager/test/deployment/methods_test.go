// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"fmt"
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
)

// HTTP method expected responses for different API endpoints
var methodResponses = map[string]map[string]int{
	"deployments": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusMethodNotAllowed,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusMethodNotAllowed,
		http.MethodPost:   http.StatusBadRequest, // 400 when body is empty
	},
	"deploymentsPerCluster": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusMethodNotAllowed,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusMethodNotAllowed,
		http.MethodPost:   http.StatusMethodNotAllowed,
	},
	"deploymentById": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusOK,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusBadRequest,
		http.MethodPost:   http.StatusMethodNotAllowed,
	},
	"deploymentsStatus": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusMethodNotAllowed,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusMethodNotAllowed,
		http.MethodPost:   http.StatusMethodNotAllowed,
	},
	"deploymentClusters": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusMethodNotAllowed,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusMethodNotAllowed,
		http.MethodPost:   http.StatusMethodNotAllowed,
	},
}

// TestListDeploymentsMethod tests the list deployments method
func (s *TestSuite) TestListDeploymentsMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments", deploymentRESTServerUrl)
	testEndpointMethods(s, url, methodResponses["deployments"], "list deployments")
}

// TestListDeploymentsPerClusterMethod tests the list deployments per cluster method
func (s *TestSuite) TestListDeploymentsPerClusterMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/clusters/%s",
		deploymentRESTServerUrl, utils.TestClusterID)
	testEndpointMethods(s, url, methodResponses["deploymentsPerCluster"], "list deployments per cluster")
}

// TestGetDeleteDeploymentMethod tests the get and delete deployment method
func (s *TestSuite) TestGetDeleteDeploymentMethod() {
	deployID, retCode, err := utils.StartDeployment(admclient, dpConfigName, "targeted", 10)
	s.Equal(retCode, http.StatusOK)
	s.NoError(err)
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s", deploymentRESTServerUrl, deployID)
	testEndpointMethods(s, url, methodResponses["deploymentById"], "get and delete deployment")
}

// TestGetDeploymentsStatusMethod tests the get deployments status method
func (s *TestSuite) TestGetDeploymentsStatusMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/summary/deployments_status", deploymentRESTServerUrl)
	testEndpointMethods(s, url, methodResponses["deploymentsStatus"], "get deployments status")
}

// TestListDeploymentClustersMethod tests the list deployment clusters method
func (s *TestSuite) TestListDeploymentClustersMethod() {
	deployID, retCode, err := utils.StartDeployment(admclient, dpConfigName, "targeted", 10)
	s.Equal(retCode, http.StatusOK)
	s.NoError(err)
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s/clusters",
		deploymentRESTServerUrl, deployID)
	testEndpointMethods(s, url, methodResponses["deploymentClusters"], "list deployment clusters")
}

// Helper function to test HTTP methods on an endpoint
func testEndpointMethods(s *TestSuite, url string, methodMap map[string]int, description string) {
	for method, expectedStatus := range methodMap {
		res, err := utils.CallMethod(url, method, token, projectID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)
		s.T().Logf("%s method: %s (%d)\n", description, method, res.StatusCode)
	}
}
