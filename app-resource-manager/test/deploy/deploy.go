// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deploy

import (
	// "fmt"
	"time"

	// "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/auth"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

const (
	wordpressAppName             = "nginx"
	wordpressDP                  = "nginx-app"
	wordpressAppVersion          = "0.1.0"
	wordpressTargetedDisplayName = "wordpress-targeted"
	TestClusterID                = "demo-ruben3"
	wordpressProfileName         = "testing-default"
	retryCount                   = 10
	retryDelay                   = 10 * time.Second
)

func CreateDeployment(admClient *restClient.ClientWithResponses) ([]*restClient.App, error) {
	// Delete existing "wordpress" deployment if it exists
	// s.T().Log("Attempting to delete existing 'wordpress' deployment...")
	err := deleteAndRetryUntilDeleted(admClient, wordpressTargetedDisplayName, retryCount, retryDelay)
	if err != nil {
		return []*restClient.App{}, err
	}
	// s.NoError(err, "Failed to delete existing deployment")

	// s.NoError(err, "Failed to delete existing deployment")
	// s.T().Log("'wordpress' deployment deletion initiated.")

	// Create a new "wordpress" deployment
	// s.T().Log("Creating a new 'wordpress' deployment...")
	err = createTargetedDeployment(admClient, CreateDeploymentParams{
		ClusterID:      TestClusterID,
		DpName:         wordpressDP,
		AppName:        wordpressAppName,
		AppVersion:     wordpressAppVersion,
		DisplayName:    wordpressTargetedDisplayName,
		ProfileName:    wordpressProfileName,
		DeploymentType: "targeted",
	})
	if err != nil {
		return []*restClient.App{}, err
	}
	// s.NoError(err, "Failed to delete existing deployment")

	// s.NoError(err, "Failed to create 'wordpress' deployment")
	// s.T().Log("'wordpress' deployment creation initiated.")

	// Wait for the deployment to reach "Running" status
	// s.T().Log("Waiting for 'wordpress' deployment to reach 'RUNNING' status...")

	deployId, err := waitForDeploymentStatus(admClient, wordpressTargetedDisplayName, restClient.RUNNING, retryCount, retryDelay)
	if err != nil {
		return []*restClient.App{}, err
	}

	deployApps, err := getDeployApps(admClient, deployId)
	if err != nil {
		return []*restClient.App{}, err
	}

	return deployApps, nil
	// s.NoError(err, "Deployment did not reach RUNNING status")
	// s.T().Log("'wordpress' deployment is now in 'RUNNING' status.")
}
