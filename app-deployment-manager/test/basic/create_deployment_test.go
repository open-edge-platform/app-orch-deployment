// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"time"
)

// TestCreateWordpressDeployment tests creating a wordpress deployment using the REST API.
func (s *TestSuite) TestCreateWordpressDeployment() {
	// Get list of deployments and if there is a deployment with same name, delete it first
	listRes, err := s.client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	s.NoError(err)
	s.Equal(200, listRes.StatusCode())
	deployments := listRes.JSON200.Deployments
	for _, deployment := range deployments {
		s.T().Log("deployment: ", *deployment.DisplayName)
		if *deployment.DisplayName == "wordpress" {
			response, err := s.client.DeploymentServiceDeleteDeploymentWithResponse(context.TODO(), *deployment.DeployId, nil)
			s.T().Log("wordpress deployment found, deleting it")
			s.NoError(err)
			s.Equal(200, response.StatusCode())
		}
	}
	// retry list deployments to confirm deletion
	for i := 0; i < 10; i++ {
		listRes, err = s.client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
		s.NoError(err)
		s.Equal(200, listRes.StatusCode())
		deployments = listRes.JSON200.Deployments
		found := false
		for _, deployment := range deployments {
			if *deployment.DisplayName == "wordpress" {
				found = true
				break
			}
		}
		if !found {
			break
		}
		time.Sleep(5 * time.Second)
	}

	deploymentType := "targeted"
	disployName := "wordpress"
	profileName := "testing"
	targetCluster := "demo-cluster"
	targetClusters := []restClient.TargetClusters{
		{
			AppName:   &disployName,
			ClusterId: &targetCluster,
		},
	}

	reqBody := restClient.DeploymentServiceCreateDeploymentJSONRequestBody{
		AppName:        "wordpress",
		AppVersion:     "0.1.1",
		DeploymentType: &deploymentType,
		DisplayName:    &disployName,
		ProfileName:    &profileName,
		TargetClusters: &targetClusters,
	}
	createRes, err := s.client.DeploymentServiceCreateDeploymentWithResponse(context.TODO(), reqBody)
	s.NoError(err)
	s.Equal(200, createRes.StatusCode())

}
