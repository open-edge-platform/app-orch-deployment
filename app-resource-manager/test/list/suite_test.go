// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package basic is a suite of basic functionality tests for the ADM service
package list

import (
	"context"
	"fmt"
	"net/http"

	"testing"

	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/auth"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"github.com/stretchr/testify/suite"
)

const (
	ArmRestAddress         = "app-resource-manager-rest-proxy:8081/"
	AdmRestAddress         = "app-deployment-api-rest-proxy:8081/"
	RestAddressPortForward = "127.0.0.1"
	KeycloakServer         = "keycloak.kind.internal"

	AdmPortForwardRemote = "8081"
	ArmPortForwardRemote = "8082"
)

type TestSuite struct {
	suite.Suite
	KeycloakServer          string
	ResourceRESTServerUrl   string
	DeploymentRESTServerUrl string
	token                   string
	projectID               string
	deployApps              []*admClient.App
	AdmClient               *admClient.ClientWithResponses
	ArmClient               *armClient.ClientWithResponses
}

// SetupTest sets up for each test
func (s *TestSuite) SetupTest() {
	var err error
	s.KeycloakServer = KeycloakServer
	s.DeploymentRESTServerUrl = AdmRestAddress
	s.ResourceRESTServerUrl = ArmRestAddress
	s.token = auth.SetUpAccessToken(s.T(), s.KeycloakServer)
	s.DeploymentRESTServerUrl = fmt.Sprintf("http://%s:%s", RestAddressPortForward, AdmPortForwardRemote)
	s.projectID, err = auth.GetProjectID(context.TODO())
	s.NoError(err)

	s.AdmClient, err = admClient.NewClientWithResponses(s.DeploymentRESTServerUrl, admClient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		auth.AddRestAuthHeader(req, s.token, s.projectID)
		return nil
	}))
	s.NoError(err)

	s.ResourceRESTServerUrl = fmt.Sprintf("http://%s:%s", RestAddressPortForward, ArmPortForwardRemote)

	s.ArmClient, err = armClient.NewClientWithResponses(s.ResourceRESTServerUrl, armClient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		auth.AddRestAuthHeader(req, s.token, s.projectID)
		return nil
	}))
	s.NoError(err)

	// todo move ADM deployment to TestTestSuite to only deploy once for whole test suite
	deployApps, err := deploy.CreateDeployment(s.AdmClient)
	s.NoError(err)
	s.NotEmpty(deployApps)
	s.T().Log("successfully deployed app\n")
}

func TestTestSuite(t *testing.T) {
	portForwardCmd, err := utils.BringUpPortForward()
	if err != nil {
		t.Errorf("error: %v", err)
	}

	suite.Run(t, new(TestSuite))

	utils.TearDownPortForward(portForwardCmd)
}
