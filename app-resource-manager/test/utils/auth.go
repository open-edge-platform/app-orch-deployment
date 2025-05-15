// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package auth contains utilities for keycloak authentication
package utils

import (
	"context"
	"encoding/json"
	"fmt"

	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"io"
	"net/http"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
)

func SetUpAccessToken(server string, username string, password string) (string, error) {
	c := &http.Client{
		Transport: &http.Transport{},
	}
	data := url.Values{}
	data.Set("client_id", "system-client")
	data.Set("username", username)
	data.Set("password", password)
	data.Set("grant_type", "password")
	url := "https://" + server + "/realms/master/protocol/openid-connect/token"
	req, err := http.NewRequest(http.MethodPost,
		url,
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("error from keycloak: %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("error from keycloak: %v", err)
	}
	if resp == nil {
		return "", fmt.Errorf("no response from keycloak: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawTokenData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	tokenData := map[string]any{}
	err = json.Unmarshal(rawTokenData, &tokenData)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling token data: %v", err)
	}

	accessToken := tokenData["access_token"].(string)
	if accessToken == "" {
		return "", fmt.Errorf("access token is empty")
	}
	return accessToken, nil
}

func AddRestAuthHeader(req *http.Request, token string, projectID string) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Activeprojectid", projectID)
}

func GetProjectID(ctx context.Context) (string, error) {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s). Error: %w", SampleProject, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s). Error: %w", SampleProject, err)
	}

	org := configNode.Orgs(SampleOrg)
	if org == nil {
		fmt.Printf("org %s does not exist.\n", SampleOrg)
		return "", nil
	}

	folder := org.Folders("default")
	if folder == nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s). Error: %w", SampleProject, err)
	}

	project := folder.Projects(SampleProject)
	projectStatus, err := project.GetProjectStatus(ctx)
	if projectStatus == nil || err != nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s). Error: %w", SampleProject, err)
	}

	return projectStatus.UID, nil
}
