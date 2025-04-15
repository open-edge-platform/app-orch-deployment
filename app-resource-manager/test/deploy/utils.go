// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

func ptr[T any](v T) *T {
	return &v
}

func deleteDeployment(client *restClient.ClientWithResponses, deployID string) error {
	resp, err := client.DeploymentServiceDeleteDeploymentWithResponse(context.TODO(), deployID, nil)
	if err != nil || resp.StatusCode() != 200 {
		return fmt.Errorf("failed to delete deployment: %v, status: %d", err, resp.StatusCode())
	}
	return nil
}

func deploymentExists(deployments []restClient.Deployment, displayName string) bool {
	for _, d := range deployments {
		if *d.DisplayName == displayName {
			return true
		}
	}
	return false
}

func getDeploymentPerCluster(client *restClient.ClientWithResponses) ([]restClient.DeploymentInstancesCluster, error) {
	resp, err := client.DeploymentServiceListDeploymentsPerClusterWithResponse(context.TODO(), TestClusterID, nil)
	if err != nil || resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to list deployment cluster: %v, status: %d", err, resp.StatusCode())
	}

	return resp.JSON200.DeploymentInstancesCluster, nil
}

func getDeployments(client *restClient.ClientWithResponses) ([]restClient.Deployment, error) {
	resp, err := client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	if err != nil || resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to list deployments: %v, status: %d", err, resp.StatusCode())
	}

	return resp.JSON200.Deployments, nil
}

func waitForDeploymentStatus(client *restClient.ClientWithResponses, displayName string, status restClient.DeploymentStatusState, retries int, delay time.Duration) (string, error) {
	for range retries {
		if deployments, err := getDeployments(client); err == nil {
			for _, d := range deployments {
				if *d.DisplayName == displayName && *d.Status.State == status {
					return *d.DeployId, nil
				}
			}
		}
		fmt.Printf("Waiting for deployment %s to reach status %s...\n", displayName, status)
		time.Sleep(delay)
	}

	return "", fmt.Errorf("deployment %s did not reach status %s after %d retries", displayName, status, retries)
}

func getDeployApps(client *restClient.ClientWithResponses, deployID string) ([]*restClient.App, error) {
	if deployments, err := getDeploymentPerCluster(client); err == nil {
		for _, d := range deployments {
			if *d.DeploymentUid == deployID {
				apps := make([]*restClient.App, len(*d.Apps))
				for i, app := range *d.Apps {
					apps[i] = &app
				}
				return apps, nil
			}
		}
	}

	return []*restClient.App{}, fmt.Errorf("did not find deployment id %s", deployID)
}

func findDeploymentIDByDisplayName(client *restClient.ClientWithResponses, displayName string) (string, error) {
	if deployments, err := getDeployments(client); err == nil {
		for _, d := range deployments {
			if *d.DisplayName == displayName {
				return *d.DeployId, nil
			}
		}
	}
	return "", fmt.Errorf("deployment %s not found", displayName)
}

func deleteDeploymentByDisplayName(client *restClient.ClientWithResponses, displayName string) error {
	if deployID, err := findDeploymentIDByDisplayName(client, displayName); err == nil {
		return deleteDeployment(client, deployID)
	}

	return fmt.Errorf("deployment %s not found", displayName)
}

type CreateDeploymentParams struct {
	DpName         string
	AppName        string
	AppVersion     string
	DisplayName    string
	ProfileName    string
	ClusterID      string
	Labels         *map[string]string
	DeploymentType string
}

func createTargetedDeployment(client *restClient.ClientWithResponses, params CreateDeploymentParams) error {
	reqBody := restClient.DeploymentServiceCreateDeploymentJSONRequestBody{
		AppName:        params.DpName,
		AppVersion:     params.AppVersion,
		DeploymentType: ptr(params.DeploymentType),
		DisplayName:    ptr(params.DisplayName),
		ProfileName:    ptr(params.ProfileName),
	}

	if params.ClusterID != "" {
		reqBody.TargetClusters = &[]restClient.TargetClusters{
			{
				AppName:   ptr(params.AppName),
				ClusterId: ptr(params.ClusterID),
			},
		}
	} else if params.Labels != nil {
		reqBody.TargetClusters = &[]restClient.TargetClusters{
			{
				AppName: ptr(params.AppName),
				Labels:  params.Labels,
			},
		}
	}

	createRes, err := client.DeploymentServiceCreateDeploymentWithResponse(context.TODO(), reqBody)

	if err != nil || createRes.StatusCode() != 200 {
		return fmt.Errorf("failed to create deployment: %v, status: %d", err, createRes.StatusCode())
	}
	return nil
}

func deleteAndRetryUntilDeleted(client *restClient.ClientWithResponses, displayName string, retries int, delay time.Duration) error {
	// Attempt to delete the deployment
	if err := deleteDeploymentByDisplayName(client, displayName); err != nil {
		return fmt.Errorf("initial deletion failed: %v", err)
	}

	// Retry until the deployment is confirmed deleted
	for i := 0; i < retries; i++ {
		if deployments, err := getDeployments(client); err == nil && !deploymentExists(deployments, displayName) {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("deployment %s not deleted after %d retries", displayName, retries)
}
