// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"

	"testing"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"github.com/stretchr/testify/suite"
)

var (
	deployID                string
	token                   string
	projectID               string
	deploymentRESTServerUrl string
	deployApps              []*restClient.App
	admclient               *restClient.ClientWithResponses
)

const (
	RestAddressPortForward = "127.0.0.1"
	KeycloakServer         = "keycloak.kind.internal"

	AdmPortForwardRemote = "8081"
	dpDisplayName        = "nginx-deployment-container"
	dpConfigName         = "nginx"
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	AdmClient *restClient.ClientWithResponses
}

// SetupTest sets up for each test
func (s *TestSuite) SetupTest() {
	s.AdmClient = admclient

	s.NotEmpty(deployApps)
	s.NotEmpty(deployID)
}

func TestDeploymentSuite(t *testing.T) {
	portForwardCmd, err := utils.BringUpPortForward()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer utils.TearDownPortForward(portForwardCmd)

	token, err = utils.SetUpAccessToken(KeycloakServer)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	projectID, err = utils.GetProjectID(context.TODO())
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	deploymentRESTServerUrl = fmt.Sprintf("http://%s:%s", RestAddressPortForward, AdmPortForwardRemote)
	admclient, err = utils.CreateAdmClient(deploymentRESTServerUrl, token, projectID)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	deployApps, err = deploy.CreateDeployment(admclient, dpConfigName, dpDisplayName, 10)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	deployID = deploy.FindDeploymentIDByDisplayName(admclient, dpDisplayName)
	if deployID == "" {
		t.Fatalf("error: %v", fmt.Errorf("deployment %s not found", dpDisplayName))
	}

	suite.Run(t, new(TestSuite))
}
