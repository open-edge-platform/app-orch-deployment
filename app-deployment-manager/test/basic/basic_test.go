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

// findDeploymentIDByAppName searches for a deployment by appName and returns its deployId.
func (s *TestSuite) findDeploymentIDByAppName(appName string) (string, error) {
	res, err := s.listDeployments(http.MethodGet)
	if err != nil {
		return "", fmt.Errorf("failed to list deployments: %w", err)
	}
	defer res.Body.Close()

	var responseBody map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&responseBody)
	if err != nil {
		return "", fmt.Errorf("failed to parse deployments list: %w", err)
	}

	deployments, ok := responseBody["deployments"].([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format: missing 'deployments' key")
	}

	for _, deployment := range deployments {
		deploymentMap, ok := deployment.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("unexpected deployment format: not a map")
		}

		appNameField, appNameOk := deploymentMap["appName"].(string)
		if !appNameOk || appNameField == "" {
			continue // Skip invalid or missing appName
		}

		if appNameField == appName {
			deployId, deployIdOk := deploymentMap["deployId"].(string)
			if deployIdOk && deployId != "" {
			