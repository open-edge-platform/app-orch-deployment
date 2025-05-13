// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

type CreateDeploymentParams struct {
	DpName         string
	AppNames       []string
	AppVersion     string
	DisplayName    string
	ProfileName    string
	ClusterID      string
	Labels         *map[string]string
	DeploymentType string
}

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

func getDeploymentPerCluster(client *restClient.ClientWithResponses) ([]restClient.DeploymentInstancesCluster, int, error) {
	resp, err := client.DeploymentServiceListDeploymentsPerClusterWithResponse(context.TODO(), TestClusterID, nil)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return nil, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return nil, resp.StatusCode(), fmt.Errorf("failed to list deployment cluster: %v", string(resp.Body))
	}

	return resp.JSON200.DeploymentInstancesCluster, resp.StatusCode(), nil
}

func getDeployments(client *restClient.ClientWithResponses) ([]restClient.Deployment, int, error) {
	resp, err := client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return nil, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return nil, resp.StatusCode(), fmt.Errorf("failed to list deployments: %v", string(resp.Body))
	}

	return resp.JSON200.Deployments, resp.StatusCode(), nil
}

func waitForDeploymentStatus(client *restClient.ClientWithResponses, displayName string, status restClient.DeploymentStatusState, retries int, delay time.Duration) (string, error) {
	currState := "UNKNOWN"
	for range retries {
		deployments, retCode, err := getDeployments(client)
		if err != nil || retCode != 200 {
			return "", fmt.Errorf("failed to get deployments: %v", err)
		}

		for _, d := range deployments {
			// In case there's several deployments only use the one with the same display name
			if *d.DisplayName == displayName {
				currState = string(*d.Status.State)
			}

			if *d.DisplayName == displayName && currState == string(status) {
				fmt.Printf("Waiting for deployment %s state %s ---> %s\n", displayName, currState, status)
				return *d.DeployId, nil
			}
		}

		fmt.Printf("Waiting for deployment %s state %s ---> %s\n", displayName, currState, status)
		time.Sleep(delay)
	}

	return "", fmt.Errorf("deployment %s did not reach status %s after %d retries", displayName, status, retries)
}

func getDeployApps(client *restClient.ClientWithResponses, deployID string) ([]*restClient.App, error) {
	deployments, retCode, err := getDeploymentPerCluster(client)
	if err != nil || retCode != 200 {
		return []*restClient.App{}, fmt.Errorf("failed to get deployments: %v", err)
	}

	for _, d := range deployments {
		if *d.DeploymentUid == deployID {
			apps := make([]*restClient.App, len(*d.Apps))
			for i, app := range *d.Apps {
				apps[i] = &app
			}
			return apps, nil
		}
	}

	return []*restClient.App{}, fmt.Errorf("did not find deployment id %s", deployID)
}

func findDeploymentIDByDisplayName(client *restClient.ClientWithResponses, displayName string) string {
	deployments, retCode, err := getDeployments(client)
	if err != nil || retCode != 200 {
		return ""
	}

	for _, d := range deployments {
		if *d.DisplayName == displayName {
			return *d.DeployId
		}
	}

	return ""
}

func deleteDeploymentByDisplayName(client *restClient.ClientWithResponses, displayName string) error {
	if deployID := findDeploymentIDByDisplayName(client, displayName); deployID != "" {
		err := deleteDeployment(client, deployID)
		if err != nil {
			return fmt.Errorf("failed to delete deployment %s: %v", displayName, err)
		}
		fmt.Printf("Deployment %s deleted\n", displayName)
		return nil
	}

	fmt.Printf("Deployment %s not found for deletion\n", displayName)
	return nil
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
		var targetClusters []restClient.TargetClusters
		for _, v := range *ptr(params.AppNames) {
			targetClusters = append(targetClusters, restClient.TargetClusters{
				AppName:   ptr(v),
				ClusterId: ptr(params.ClusterID),
			})
		}
		reqBody.TargetClusters = &targetClusters
	} else if params.Labels != nil {
		var targetClusters []restClient.TargetClusters
		for _, v := range *ptr(params.AppNames) {
			targetClusters = append(targetClusters, restClient.TargetClusters{
				AppName: ptr(v),
				Labels:  params.Labels,
			})
		}
		reqBody.TargetClusters = &targetClusters
	}

	createRes, err := client.DeploymentServiceCreateDeploymentWithResponse(context.TODO(), reqBody)
	if err != nil || createRes.StatusCode() != 200 {
		if createRes.Body != nil {
			fmt.Printf("failed to create deployment: %s\n", string(createRes.Body))
		}
		return fmt.Errorf("failed to create deployment: %v, status: %d", err, createRes.StatusCode())
	}
	return nil
}

func DeleteAndRetryUntilDeleted(client *restClient.ClientWithResponses, displayName string, retries int, delay time.Duration) error {
	// Attempt to delete the deployment
	if err := deleteDeploymentByDisplayName(client, displayName); err != nil {
		return fmt.Errorf("initial deletion failed: %v", err)
	}

	// Retry until the deployment is confirmed deleted
	for range retries {
		deployments, retCode, err := getDeployments(client)
		if err != nil || retCode != 200 {
			return fmt.Errorf("failed to get deployments: %v", err)
		}

		if !deploymentExists(deployments, displayName) {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("deployment %s not deleted after %d retries", displayName, retries)
}
