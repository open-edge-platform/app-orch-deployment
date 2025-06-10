// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package methods

import (
	"fmt"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/portforwarding"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
	"net/http"
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
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments", s.deploymentRESTServerUrl),
			methodMap:   methodResponses["deployments"],
			description: "list deployments",
		},
		{
			name:        "ListDeploymentsPerCluster",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/clusters/%s", s.deploymentRESTServerUrl, types.TestClusterID),
			methodMap:   methodResponses["deploymentsPerCluster"],
			description: "list deployments per cluster",
		},
		{
			name:        "GetAndDeleteDeployment",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s", s.deploymentRESTServerUrl, "%s"),
			methodMap:   methodResponses["deploymentById"],
			description: "get and delete deployment",
			setup: func() string {
				deploymentReq := deploymentutils.StartDeploymentRequest{
					AdmClient:         s.AdmClient,
					DpPackageName:     deploymentutils.AppNginx,
					DeploymentType:    deploymentutils.DeploymentTypeTargeted,
					DeploymentTimeout: deploymentutils.DeploymentTimeout,
					DeleteTimeout:     deploymentutils.DeleteTimeout,
					TestName:          "GetAndDeleteDeployment",
				}
				deployID, retCode, err := deploymentutils.StartDeployment(deploymentReq)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				return deployID
			},
		},
		{
			name:        "GetDeploymentsStatus",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/summary/deployments_status", s.deploymentRESTServerUrl),
			methodMap:   methodResponses["deploymentsStatus"],
			description: "get deployments status",
		},
		{
			name:        "ListDeploymentClusters",
			url:         fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s/clusters", s.deploymentRESTServerUrl, "%s"),
			methodMap:   methodResponses["deploymentClusters"],
			description: "list deployment clusters",
			setup: func() string {
				deploymentReq := deploymentutils.StartDeploymentRequest{
					AdmClient:         s.AdmClient,
					DpPackageName:     deploymentutils.AppNginx,
					DeploymentType:    deploymentutils.DeploymentTypeTargeted,
					DeploymentTimeout: deploymentutils.DeploymentTimeout,
					DeleteTimeout:     deploymentutils.DeleteTimeout,
					TestName:          "ListDeploymentClusters",
				}
				deployID, retCode, err := deploymentutils.StartDeployment(deploymentReq)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				return deployID
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			url := tc.url
			var deployID string
			if tc.setup != nil {
				deployID = tc.setup()
				url = fmt.Sprintf(tc.url, deployID)
			}
			testEndpointMethods(s, url, tc.methodMap, tc.description)
		})
	}
}

// Helper function to test HTTP methods on an endpoint
func testEndpointMethods(s *TestSuite, url string, methodResponses []MethodResponse, description string) {
	for _, mr := range methodResponses {
		res, err := portforwarding.CallMethod(url, mr.Method, s.token, s.projectID)
		s.NoError(err)
		s.Equal(mr.ExpectedCode, res.StatusCode)
		s.T().Logf("%s method: %s (%d)\n", description, mr.Method, res.StatusCode)
	}
}
