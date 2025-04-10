// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"time"
)

const (
	worldpressAppName     = "wordpress"
	worldpressAppVersion  = "0.1.1"
	worldpressDisplayName = "wordpress"
	testClusterID         = "demo-cluster"
	wordpressProfileName  = "testing"
	retryCount            = 10
	retryDelay            = 10 * time.Second
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	// Delete existing "wordpress" deployment if it exists
	s.T().Log("Attempting to delete existing 'wordpress' deployment...")
	err := deleteAndRetryUntilDeleted(s.client, worldpressDisplayName, retryCount, retryDelay)
	s.NoError(err, "Failed to delete existing deployment")
	s.T().Log("'wordpress' deployment deletion initiated.")

	// Create a new "wordpress" deployment
	s.T().Log("Creating a new 'wordpress' deployment...")
	err = createTargetedDeployment(s.client, CreateDeploymentParams{
		ClusterID:      testClusterID,
		AppName:        worldpressAppName,
		AppVersion:     worldpressAppVersion,
		DisplayName:    worldpressDisplayName,
		ProfileName:    wordpressProfileName,
		DeploymentType: "targeted",
	})
	s.NoError(err, "Failed to create 'wordpress' deployment")
	s.T().Log("'wordpress' deployment creation initiated.")

	// Wait for the deployment to reach "Running" status
	s.T().Log("Waiting for 'wordpress' deployment to reach 'RUNNING' status...")
	err = waitForDeploymentStatus(s.client, worldpressDisplayName, restClient.RUNNING, retryCount, retryDelay)
	s.NoError(err, "Deployment did not reach RUNNING status")
	s.T().Log("'wordpress' deployment is now in 'RUNNING' status.")
}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	// Delete existing "wordpress" deployment if it exists
	s.T().Log("Attempting to delete existing 'wordpress' deployment...")
	err := deleteAndRetryUntilDeleted(s.client, worldpressDisplayName, retryCount, retryDelay)
	s.NoError(err, "Failed to delete existing deployment")
	s.T().Log("'wordpress' deployment deletion initiated.")

	// Create a new "wordpress" deployment
	s.T().Log("Creating a new 'wordpress' deployment...")
	err = createTargetedDeployment(s.client, CreateDeploymentParams{
		AppName:        worldpressAppName,
		AppVersion:     worldpressAppVersion,
		DisplayName:    worldpressDisplayName,
		ProfileName:    wordpressProfileName,
		DeploymentType: "auto-scaling",
		Labels: &map[string]string{
			"color": "blue",
		},
	})
	s.NoError(err, "Failed to create 'wordpress' deployment")
	s.T().Log("'wordpress' deployment creation initiated.")

	// Wait for the deployment to reach "Running" status
	s.T().Log("Waiting for 'wordpress' deployment to reach 'RUNNING' status...")
	err = waitForDeploymentStatus(s.client, worldpressDisplayName, restClient.RUNNING, retryCount, retryDelay)
	s.NoError(err, "Deployment did not reach RUNNING status")
	s.T().Log("'wordpress' deployment is now in 'RUNNING' status.")
}
