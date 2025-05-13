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

const InvalidJWT = "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.ey" +
	"JzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRt" +
	"aW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.J5W09-rNx0pt5_HBiy" +
	"dR-vOluS6oD-RpYNa8PVWwMcBDQSXiw6-EPW8iSsalXPspGj3ouQjA" +
	"nOP_4-zrlUUlvUIt2T79XyNeiKuooyIFvka3Y5NnGiOUBHWvWcWp4R" +
	"cQFMBrZkHtJM23sB5D7Wxjx0-HFeNk-Y3UJgeJVhg5NaWXypLkC4y0" +
	"ADrUBfGAxhvGdRdULZivfvzuVtv6AzW6NRuEE6DM9xpoWX_4here-y" +
	"vLS2YPiBTZ8xbB3axdM99LhES-n52lVkiX5AWg2JJkEROZzLMpaacA" +
	"_xlbUz_zbIaOaoqk8gB5oO7kI6sZej3QAdGigQy-hXiRnW_L98d4GQ"

const (
	SampleOrg     = "sample-org"
	SampleProject = "sample-project"
	DefaultPass   = "ChangeMeOn1stLogin!"
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
