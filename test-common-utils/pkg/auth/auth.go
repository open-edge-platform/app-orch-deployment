// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package auth contains utilities for keycloak authentication
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
	"os"
	"strconv"

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

func SetUpAccessToken(server string) (string, error) {
	c := &http.Client{
		Transport: &http.Transport{},
	}
	data := url.Values{}
	data.Set("client_id", "system-client")
	data.Set("username", types.SampleUsername)
	data.Set("password", types.KCPass)
	data.Set("grant_type", "password")
	url := "https://" + server + "/realms/master/protocol/openid-connect/token"
	req, err := http.NewRequest(http.MethodPost,
		url,
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("error from keycloak: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	fmt.Printf("Authenticating with keycloak as user: %s\n", types.SampleUsername)

	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("error from keycloak: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("no response from keycloak: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawTokenData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("keycloak auth failed for user %s: status %d, body: %s", types.SampleUsername, resp.StatusCode, string(rawTokenData))
	}

	tokenData := map[string]any{}
	err = json.Unmarshal(rawTokenData, &tokenData)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling token data: %w", err)
	}

	accessToken, ok := tokenData["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access token not found in keycloak response: %s", string(rawTokenData))
	}
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
		return "", fmt.Errorf("\nerror retrieving the project (%s): %w", types.SampleProject, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s): %w", types.SampleProject, err)
	}

	org := configNode.Orgs(types.SampleOrg)
	if org == nil {
		fmt.Printf("org %s does not exist.\n", types.SampleOrg)
		return "", nil
	}

	folder := org.Folders("default")
	if folder == nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s): %w", types.SampleProject, err)
	}

	project := folder.Projects(types.SampleProject)
	projectStatus, err := project.GetProjectStatus(ctx)
	if projectStatus == nil || err != nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s): %w", types.SampleProject, err)
	}

	return projectStatus.UID, nil
}

func GetKeycloakServer() string {
	orchDomain := GetOrchDomain()
	return fmt.Sprintf("keycloak.%s", orchDomain)
}

func GetOrchDomain() string {
	autoCert, err := strconv.ParseBool(os.Getenv("AUTO_CERT"))
	orchDomain := os.Getenv("ORCH_DOMAIN")
	if err != nil || !autoCert || orchDomain == "" {
		orchDomain = "kind.internal"
	}
	return orchDomain
}
