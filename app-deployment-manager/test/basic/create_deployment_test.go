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
	res, err := s.client.DeploymentServiceCreateDeploymentWithResponse(context.TODO(), reqBody)
	s.NoError(err)
	s.Equal(200, res.StatusCode())

	// Check list of deployments until wordpress deployment status become running
	for {
		res, err := s.client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
		s.NoError(err)
		s.Equal(200, res.StatusCode())
		if len(res.JSON200.Deployments) > 0 {
			for _, deployment := range res.JSON200.Deployments {
				if *deployment.DisplayName == disployName && *deployment.Status.State == "Running" {
					s.T().Logf("Deployment %s is running", disployName)
					return
				}
			}
		}
		time.Sleep(5 * time.Second)
	}

}
