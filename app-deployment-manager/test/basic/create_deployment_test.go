// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
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
)

// TestCreateWordpressDeployment tests creating a wordpress deployment using the REST API.
func (s *TestSuite) TestCreateWordpressDeployment() {
	// Delete existing "wordpress" deployment if it exists
	if deployments, err := s.getDeployments(); err == nil {
		for _, deployment := range deployments {
			if *deployment.DisplayName == worldpressDisplayName {
				s.deleteDeployment(*deployment.DeployId)
			}
		}
	}

	// Confirm deletion of "wordpress" deployment
	s.retryUntilDeleted("wordpress", 10, 5*time.Second)

	// Create a new "wordpress" deployment
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

func (s *TestSuite) getDeployments() ([]restClient.Deployment, error) {
	listRes, err := s.client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	if err != nil || listRes.StatusCode() != 200 {
		return nil, err
	}
	return listRes.JSON200.Deployments, nil
}

func (s *TestSuite) deleteDeployment(deployId string) {
	response, err := s.client.DeploymentServiceDeleteDeploymentWithResponse(context.TODO(), deployId, nil)
	s.NoError(err)
	s.Equal(200, response.StatusCode())
}

func (s *TestSuite) retryUntilDeleted(displayName string, retries int, delay time.Duration) {
	for i := 0; i < retries; i++ {
		if deployments, err := s.getDeployments(); err == nil {
			found := false
			for _, deployment := range deployments {
				if *deployment.DisplayName == displayName {
					found = true
					break
				}
			}
			if !found {
				s.T().Logf("%s deployment deleted", displayName)
				return
			}
		}
		time.Sleep(delay)
	}
}

func ptr[T any](v T) *T {
	return &v
}
