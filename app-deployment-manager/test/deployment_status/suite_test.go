// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/portforwarding"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
	"github.com/stretchr/testify/suite"
	"os/exec"
	"testing"
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	AdmClient               *restClient.ClientWithResponses
	PortForwardCmd          map[string]*exec.Cmd
	deploymentRESTServerUrl string
	token                   string
	projectID               string
}

func (s *TestSuite) SetupSuite() {
	var err error
	s.PortForwardCmd, err = portforwarding.StartPortForwarding()
	if err != nil {
		s.T().Fatalf("failed to bring up port forward: %v", err)
	}

	s.token, err = auth.SetUpAccessToken(auth.GetKeycloakServer())
	if err != nil {
		s.T().Fatalf("failed to setup access token: %v", err)
	}

	s.projectID, err = auth.GetProjectID(context.TODO())
	if err != nil {
		s.T().Fatalf("failed to get project id: %v", err)
	}

	s.deploymentRESTServerUrl = fmt.Sprintf("http://%s:%s", types.RestAddressPortForward, types.AdmPortForwardRemote)
	s.AdmClient, err = clients.CreateAdmClient(s.deploymentRESTServerUrl, s.token, s.projectID)
	if err != nil {
		s.T().Fatalf("failed to create client: %v", err)
	}
}

func TestDeploymentStatusSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// TearDownSuite cleans up after the entire test suite
func (s *TestSuite) TearDownSuite() {

}
