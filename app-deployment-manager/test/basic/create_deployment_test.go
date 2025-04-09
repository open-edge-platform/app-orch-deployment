// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/auth"
)

// Add a test for creating a wordpress deployment using REST API

// createDeployment sends a POST request to create a deployment and returns the response.
func (s *TestSuite) createDeployment(reqBody map[string]interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments", s.DeploymentRESTServerUrl)
	fmt.Println("Create Deployment")

	bodyReader := bytes.NewReader(s.MarshalRequestBody(reqBody)) // Wrap []byte in bytes.NewReader
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
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

// MarshalRequestBody marshals a map into a JSON byte slice.
func (s *TestSuite) MarshalRequestBody(reqBody map[string]interface{}) []byte {
	body, err := json.Marshal(reqBody)
	s.Require().NoError(err, "Failed to marshal request body")
	return body
}

// deleteDeployment sends a DELETE request to remove a deployment by name and waits for it to be fully deleted.
func (s *TestSuite) deleteDeployment(appName string) error {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s", s.DeploymentRESTServerUrl, appName)
	fmt.Println("Delete Deployment")

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	auth.AddRestAuthHeader(req, s.token, s.projectID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	// Poll to ensure the deployment is fully deleted
	for i := 0; i < 10; i++ { // Retry up to 10 times
		listRes, err := s.listDeployments(http.MethodGet)
		if err != nil {
			return err
		}
		defer listRes.Body.Close()

		var deployments []map[string]interface{}
		err = json.NewDecoder(listRes.Body).Decode(&deployments)
		if err != nil {
			return err
		}

		found := false
		for _, deployment := range deployments {
			if deployment["appName"] == appName {
				found = true
				break
			}
		}

		if !found {
			return nil // Deployment successfully deleted
		}

		time.Sleep(1 * time.Second) // Wait before retrying
	}

	return fmt.Errorf("deployment '%s' was not fully deleted", appName)
}

// TestCreateWordpressDeployment tests creating a wordpress deployment using the REST API.
func (s *TestSuite) TestCreateWordpressDeployment() {
	reqBody := map[string]interface{}{
		"appName":        "wordpress",
		"appVersion":     "0.1.1",
		"deploymentType": "targeted",
		"displayName":    "wordpress",
		"networkName":    "",
		"overrideValues": []interface{}{},
		"profileName":    "testing",
		"publisherName":  "intel",
		"serviceExports": []interface{}{},
		"targetClusters": []map[string]interface{}{
			{
				"appName":   "wordpress",
				"clusterId": "demo-cluster",
			},
		},
	}

	// Ensure any existing deployment with the same name is deleted
	err := s.deleteDeployment("wordpress")
	s.Require().NoError(err, "Failed to delete existing deployment")

	// Call the helper method to create the deployment
	res, err := s.createDeployment(reqBody)
	s.Require().NoError(err, "Failed to send POST request")
	s.Require().Equal(201, res.StatusCode, "Expected status code 201 for successful deployment creation")

	// Verify the deployment exists by listing deployments
	listRes, err := s.listDeployments(http.MethodGet)
	s.Require().NoError(err, "Failed to send GET request")
	s.Require().Equal("200 OK", listRes.Status, "Expected status code 200 for listing deployments")

	// Parse the response body
	var deployments []map[string]interface{}
	err = json.NewDecoder(listRes.Body).Decode(&deployments)
	s.Require().NoError(err, "Failed to parse deployments list")

	// Check if the created deployment exists in the list
	found := false
	for _, deployment := range deployments {
		if deployment["appName"] == "wordpress" && deployment["displayName"] == "wordpress" {
			found = true
			break
		}
	}
	s.Require().True(found, "Deployment 'wordpress' not found in the list")
}
