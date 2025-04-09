package basic

import (
	"context"
	"time"

	"github.com/avast/retry-go"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

const (
	worldpressAppName     = "wordpress"
	worldpressAppVersion  = "0.1.1"
	worldpressDisplayName = "wordpress"
	testClusterID         = "demo-cluster"
	retryCount            = 10
	retryDelay            = 5 * time.Second
)

func (s *TestSuite) TestCreateWordpressDeployment() {
	// Delete existing "wordpress" deployment if it exists
	s.deleteExistingDeployment(worldpressDisplayName)

	// Confirm deletion of "wordpress" deployment
	s.retryUntilDeleted(worldpressDisplayName)

	// Create a new "wordpress" deployment
	s.createDeployment()

	// Wait for the deployment to reach "Running" status
	s.waitForDeploymentStatus(worldpressDisplayName, restClient.RUNNING)
}

func (s *TestSuite) deleteExistingDeployment(displayName string) {
	if deployments, err := s.getDeployments(); err == nil {
		for _, deployment := range deployments {
			if *deployment.DisplayName == displayName {
				s.deleteDeployment(*deployment.DeployId)
			}
		}
	}
}

func (s *TestSuite) createDeployment() {
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

func (s *TestSuite) waitForDeploymentStatus(displayName string, status restClient.DeploymentStatusState) {
	err := retry.Do(
		func() error {
			deployments, err := s.getDeployments()
			if err != nil {
				return err
			}
			for _, deployment := range deployments {
				if *deployment.DisplayName == displayName && *deployment.Status.State == status {
					return nil
				}
			}
			return retry.Unrecoverable(err) // Continue retrying if status is not met
		},
		retry.Attempts(uint(retryCount)),
		retry.Delay(retryDelay),
	)
	s.NoError(err)
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

func (s *TestSuite) retryUntilDeleted(displayName string) {
	err := retry.Do(
		func() error {
			deployments, err := s.getDeployments()
			if err != nil {
				return err
			}
			if s.deploymentExists(deployments, displayName) {
				return retry.Unrecoverable(err) // Continue retrying if deployment still exists
			}
			s.T().Logf("%s deployment deleted", displayName)
			return nil
		},
		retry.Attempts(uint(retryCount)),
		retry.Delay(retryDelay),
	)
	s.NoError(err)
}

func (s *TestSuite) deploymentExists(deployments []restClient.Deployment, displayName string) bool {
	for _, deployment := range deployments {
		if *deployment.DisplayName == displayName {
			return true
		}
	}
	return false
}

func ptr[T any](v T) *T {
	return &v
}
