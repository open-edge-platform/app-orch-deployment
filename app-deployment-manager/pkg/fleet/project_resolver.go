// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TenantManagerResolver implements ProjectResolver by calling the
// Tenant Manager internal (unauthenticated) REST API:
// GET /v1/internal/projects/{id}.
// This endpoint requires no JWT; access is restricted to in-cluster
// traffic via network policy, consistent with /v1/events and /v1/status.
type TenantManagerResolver struct {
	baseURL string
	client  *http.Client
}

// NewTenantManagerResolver creates a resolver that talks to the
// Tenant Manager at the given base URL (e.g.,
// "http://tenancy-manager.orch-platform.svc:8080").
func NewTenantManagerResolver(baseURL string) *TenantManagerResolver {
	return &TenantManagerResolver{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// projectResponse mirrors the Tenant Manager's ProjectResponse shape.
// Only the fields we need are decoded.
type projectResponse struct {
	Name    string `json:"name"`
	OrgName string `json:"orgName"`
}

func (r *TenantManagerResolver) ResolveProject(ctx context.Context, projectID string) (string, string, error) {
	u := fmt.Sprintf("%s/v1/internal/projects/%s", r.baseURL, url.PathEscape(projectID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("calling tenant manager: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("project %s not found in tenant manager", projectID)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("tenant manager returned %d: %s", resp.StatusCode, string(body))
	}

	var pr projectResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return "", "", fmt.Errorf("decoding response: %w", err)
	}

	if pr.OrgName == "" {
		return "", "", fmt.Errorf("project %s has no org in tenant manager", projectID)
	}

	return pr.OrgName, pr.Name, nil
}
