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
type MethodResponse struct {
	Method       string
	ExpectedCode int
}

var methodResponses = map[string][]MethodResponse{
	"deployments": {
		{http.MethodGet, http.StatusOK},
		{http.MethodDelete, http.StatusMethodNotAllowed},
		{http.MethodPatch, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusMethodNotAllowed},
		{http.MethodPost, http.StatusBadRequest}, // 400 when body is empty
	},
	"deploymentsPerCluster": {
		{http.MethodGet, http.StatusOK},
		{http.MethodDelete, http.StatusMethodNotAllowed},
		{http.MethodPatch, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusMethodNotAllowed},
		{http.MethodPost, http.StatusMethodNotAllowed},
	},
	"deploymentById": {
		{http.MethodGet, http.StatusOK},
		{http.MethodDelete, http.StatusOK},
		{http.MethodPatch, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusBadRequest},
		{http.MethodPost, http.StatusMethodNotAllowed},
	},
	"deploymentsStatus": {
		{http.MethodGet, http.StatusOK},
		{http.MethodDelete, http.StatusMethodNotAllowed},
		{http.MethodPatch, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusMethodNotAllowed},
		{http.MethodPost, http.StatusMethodNotAllowed},
	},
	"deploymentClusters": {
		{http.MethodGet, http.StatusOK},
		{http.MethodDelete, http.StatusMethodNotAllowed},
		{http.MethodPatch, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusMethodNotAllowed},
		{http.MethodPost, http.StatusMethodNotAllowed},
	},
}

// TestAPIMethods validates HTTP methods for various API endpoints
func (s *TestSuite) TestAPIMethods() {
	testCases := []struct {
		name        string
		url         string
		methodMap   []MethodResponse
		description string
		setup       func() string // Optional setup function, returns dynamic ID if needed
	}{
		{
			name:        "ListDeployments",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments", deploymentRESTServerUrl),
			methodMap:   methodResponses["deployments"],
			description: "list deployments",
		},
		{
			name:        "ListDeploymentsPerCluster",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/clusters/%s", deploymentRESTServerUrl, utils.TestClusterID),
			methodMap:   methodResponses["deploymentsPerCluster"],
			description: "list deployments per cluster",
		},
		{
			name:        "GetAndDeleteDeployment",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s", deploymentRESTServerUrl, "%s"),
			methodMap:   methodResponses["deploymentById"],
			description: "get and delete deployment",
			setup: func() string {
				deployID, retCode, err := utils.StartDeployment(admclient, dpConfigName, "targeted", 10)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				return deployID
			},
		},
		{
			name:        "GetDeploymentsStatus",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/summary/deployments_status", deploymentRESTServerUrl),
			methodMap:   methodResponses["deploymentsStatus"],
			description: "get deployments status",
		},
		{
			name:        "ListDeploymentClusters",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s/clusters", deploymentRESTServerUrl, "%s"),
			methodMap:   methodResponses["deploymentClusters"],
			description: "list deployment clusters",
			setup: func() string {
				deployID, retCode, err := utils.StartDeployment(admclient, dpConfigName, "targeted", 10)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				return deployID
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			url := tc.url
			if tc.setup != nil {
				deployID := tc.setup()
				url = fmt.Sprintf(tc.url, deployID)
			}
			testEndpointMethods(s, url, tc.methodMap, tc.description)
		})
	}
}

// Helper function to test HTTP methods on an endpoint
func testEndpointMethods(s *TestSuite, url string, methodResponses []MethodResponse, description string) {
	for _, mr := range methodResponses {
		res, err := utils.CallMethod(url, mr.Method, token, projectID)
		s.NoError(err)
		s.Equal(mr.ExpectedCode, res.StatusCode)
		s.T().Logf("%s method: %s (%d)\n", description, mr.Method, res.StatusCode)
	}
}
