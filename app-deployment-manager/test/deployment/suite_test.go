// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"

	"testing"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
	"github.com/stretchr/testify/suite"
)

var (
	token                   string
	projectID               string
	deploymentRESTServerUrl string
	admclient               *restClient.ClientWithResponses
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	AdmClient *restClient.ClientWithResponses
}

// SetupTest sets up for each test
func (s *TestSuite) SetupTest() {
	s.AdmClient = admclient

}

func TestDeploymentSuite(t *testing.T) {
	t.Parallel()
	portForwardCmd, err := utils.BringUpPortForward()
	if err != nil {
		t.Fatalf("failed to bring up port forward: %v", err)
	}
	defer utils.TearDownPortForward(portForwardCmd)

	token, err = utils.SetUpAccessToken(utils.KeycloakServer)
	if err != nil {
		t.Fatalf("failed to setup access token: %v", err)
	}

	projectID, err = utils.GetProjectID(context.TODO())
	if err != nil {
		t.Fatalf("failed to get project id: %v", err)
	}

	deploymentRESTServerUrl = fmt.Sprintf("http://%s:%s", utils.RestAddressPortForward, utils.AdmPortForwardRemote)
	admclient, err = utils.CreateClient(deploymentRESTServerUrl, token, projectID)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	suite.Run(t, new(TestSuite))
}
