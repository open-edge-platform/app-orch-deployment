// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"time"
)

const (
	worldpressAppName     = "wordpress"
	worldpressAppVersion  = "0.1.1"
	worldpressDisplayName = "wordpress"
	testClusterID         = "demo-cluster"
	retryCount            = 10
	retryDelay            = 5 * time.Second
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	// Delete existing "wordpress" deployment if it exists

	err := deleteDeploymentByDisplayName(s.client, worldpressDisplayName)
	s.NoError(err, "Failed to delete existing deployment")

	// Confirm deletion of "wordpress" deployment
	err = retryUntilDeleted(s.client, worldpressDisplayName, retryCount, retryDelay)
	s.NoError(err, "Failed to delete existing deployment")

	// Create a new "wordpress" deployment
	s.createWordpressDeployment()

	// Wait for the deployment to reach "Running" status
	err = waitForDeploymentStatus(s.client, worldpressDisplayName, restClient.RUNNING, retryCount, retryDelay)
	s.NoError(err, "Deployment did not reach RUNNING status")
}

func (s *TestSuite) createWordpressDeployment() {
	reqBody := restClient.DeploymentServiceCreateDeploymentJSONRequestBody{
		AppName:        worldpressAppName,
		AppVersion:     worldpressAppVersion,
		DeploymentType: ptr("targeted"),
		DisplayName:    ptr(worldpressDisplayName),
		ProfileName:    ptr("testing"),
		TargetClusters: &[]restClient.TargetClusters{
			{
				AppName:   ptr(worldpressAppName),
				ClusterId: ptr(testClusterID),
			},
		},
	}
	createRes, err := s.client.DeploymentServiceCreateDeploymentWithResponse(context.TODO(), reqBody)
	s.NoError(err)
	s.Equal(200, createRes.StatusCode())
}
