// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"bytes"
	"context"
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

// deleteDeployment sends a DELETE request to remove a deployment by ID and waits for it to be fully deleted.
func (s *TestSuite) deleteDeployment(deploymentID string) error {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s", s.DeploymentRESTServerUrl, deploymentID)
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

		var responseBody map[string]interface{}
		err = json.NewDecoder(listRes.Body).Decode(&responseBody)
		listRes.Body.Close() // Ensure the body is closed after decoding
		if err != nil {
			return err
		}

		deployments, ok := responseBody["deployments"].([]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format: missing 'deployments' key")
		}

		found := false
		for _, deployment := range deployments {
			deploymentMap, ok := deployment.(map[string]interface{})
			if ok && deploymentMap["id"] == deploymentID {
				found = true
				break
			}
		}

		if !found {
			return nil // Deployment successfully deleted
		}

		time.Sleep(1 * time.Second) // Wait before retrying
	}

	return fmt.Errorf("deployment with ID '%s' was not fully deleted", deploymentID)
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
	listRes, err := s.listDeployments(http.MethodGet)
	s.Require().NoError(err, "Failed to list deployments")

	var responseBody map[string]interface{}
	err = json.NewDecoder(listRes.Body).Decode(&responseBody)
	listRes.Body.Close()
	s.Require().NoError(err, "Failed to parse deployments list")

	deployments, ok := responseBody["deployments"].([]interface{})
	s.Require().True(ok, "Unexpected response format: missing 'deployments' key")

	for _, deployment := range deployments {
		deploymentMap, ok := deployment.(map[string]interface{})
		if ok && deploymentMap["appName"] == "wordpress" {
			err = s.deleteDeployment(deploymentMap["id"].(string))
			s.Require().NoError(err, "Failed to delete existing deployment")
			break
		}
	}

	// Call the helper method to create the deployment
	res, err := s.createDeployment(reqBody)
	s.Require().NoError(err, "Failed to send POST request")

	// Log the response body for debugging in case of unexpected status codes
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&responseBody)
	s.Require().NoError(err, "Failed to parse response body")
	fmt.Printf("Response Body: %+v\n", responseBody)

	s.Require().True(res.StatusCode == 200, "Expected status code 201 or 200 for successful deployment creation")

	// Verify the deployment exists by listing deployments
	listRes, err = s.listDeployments(http.MethodGet)
	s.Require().NoError(err, "Failed to send GET request")
	s.Require().Equal("200 OK", listRes.Status, "Expected status code 200 for listing deployments")

	// Parse the response body
	var listResponseBody map[string]interface{}
	err = json.NewDecoder(listRes.Body).Decode(&listResponseBody)
	s.Require().NoError(err, "Failed to parse deployments list")

	deployments, ok = listResponseBody["deployments"].([]interface{})
	s.Require().True(ok, "Unexpected response format: missing 'deployments' key")

	// Check if the created deployment exists in the list
	found := false
	for _, deployment := range deployments {
		deploymentMap, ok := deployment.(map[string]interface{})
		if ok && deploymentMap["appName"] == "wordpress" && deploymentMap["displayName"] == "wordpress" {
			found = true
			break
		}
	}
	s.Require().True(found, "Deployment 'wordpress' not found in the list")
	s.TearDownTest(context.TODO())
}
