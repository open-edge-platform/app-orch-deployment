// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/portforwarding"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"

	"testing"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
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
	portForwardCmd, err := portforwarding.StartPortForwarding()
	if err != nil {
		t.Fatalf("failed to bring up port forward: %v", err)
	}
	defer portforwarding.TearDownPortForward(portForwardCmd)

	token, err = auth.SetUpAccessToken(auth.GetKeycloakServer())
	if err != nil {
		t.Fatalf("failed to setup access token: %v", err)
	}

	projectID, err = auth.GetProjectID(context.TODO())
	if err != nil {
		t.Fatalf("failed to get project id: %v", err)
	}

	deploymentRESTServerUrl = fmt.Sprintf("http://%s:%s", types.RestAddressPortForward, types.AdmPortForwardRemote)
	admclient, err = clients.CreateAdmClient(deploymentRESTServerUrl, token, projectID)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	suite.Run(t, new(TestSuite))
}
