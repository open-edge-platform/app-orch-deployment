// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/auth"
)

// listDeployments allows the verb to be overridden, for tests related to http verb restriction
func (s *TestSuite) listDeployments(verb string) (*http.Response, error) {
	// url := "https://api.kind.internal/v1/projects/sample-project1/appdeployment/deployments"
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments", s.DeploymentRESTServerUrl)
	fmt.Println("List Deployments")

	req, err := http.NewRequest(verb, url, nil)
	if err != nil {
		return nil, err
	}

	auth.AddRestAuthHeader(req, s.token, s.projectID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, err
}

// TestRest tests basics of exercising the REST API of the catalog service.
func (s *TestSuite) TestBasics() {
	res, err := s.listDeployments(http.MethodGet)
	s.NoError(err)
	s.Equal("200 OK", res.Status)

	var responseBody map[string]interface{} // Adjust to handle object response
	err = json.NewDecoder(res.Body).Decode(&responseBody)
	s.NoError(err, "Failed to parse response body")

	_, ok := responseBody["deployments"].([]interface{}) // Check for deployments key
	s.True(ok, "Response does not contain 'deployments' key")

	s.TearDownTest(context.TODO())
}
