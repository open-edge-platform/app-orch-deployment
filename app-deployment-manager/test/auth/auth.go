// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package auth contains utilities for keycloak authentication
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"testing"
)

const (
	SampleOrg     = "sample-org"
	SampleProject = "sample-project"
)

func SetUpAccessToken(t *testing.T, server string) string {
	c := &http.Client{
		Transport: &http.Transport{},
	}
	data := url.Values{}
	data.Set("client_id", "system-client")
	data.Set("username", fmt.Sprintf("%s-edge-mgr", SampleProject))
	data.Set("password", "ChangeMeOn1stLogin!")
	data.Set("grant_type", "password")
	url := "https://" + server + "/realms/master/protocol/openid-connect/token"
	req, err := http.NewRequest(http.MethodPost,
		url,
		strings.NewReader(data.Encode()))
	assert.NoError(t, err)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.Do(req)
	assert.NoError(t, err)
	if resp == nil {
		fmt.Fprintf(os.Stderr, "No response from keycloak: %s\n", url)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	rawTokenData, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	tokenData := map[string]interface{}{}
	err = json.Unmarshal(rawTokenData, &tokenData)
	assert.NoError(t, err)

	accessToken := tokenData["access_token"].(string)
	assert.NotContains(t, accessToken, `named cookie not present`)
	return accessToken
}

func AddRestAuthHeader(req *http.Request, token string, projectID string) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Activeprojectid", fmt.Sprintf("%s", projectID))
}

func GetProjectId(ctx context.Context) (string, error) {
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
