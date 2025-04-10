// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package basic is a suite of basic functionality tests for the ADM service
package basic

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/auth"
	"github.com/stretchr/testify/suite"

	"testing"
	"time"
)

const (
	RestAddress            = "app-deployment-api-rest-proxy:8081/"
	RestAddressPortForward = "127.0.0.1"
	KeycloakServer         = "keycloak.kind.internal"

	PortForwardServiceNamespace = "orch-app"
	PortForwardService          = "svc/app-deployment-api-rest-proxy"
	PortForwardLocalPort        = "8081"
	PortForwardAddress          = "0.0.0.0"
	PortForwardRemotePort       = "8081"
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	KeycloakServer          string
	DeploymentRESTServerUrl string
	token                   string
	projectID               string
	portForwardCmd          *exec.Cmd
	client                  *restClient.ClientWithResponses
	createdDeployments      []string
}

// SetupSuite sets-up the integration tests for the ADM basic test suite
func (s *TestSuite) SetupSuite() {
	s.KeycloakServer = KeycloakServer
	s.DeploymentRESTServerUrl = RestAddress
}

// SetupTest sets up for each integration test
func (s *TestSuite) SetupTest() {
	var err error
	s.token = auth.SetUpAccessToken(s.T(), s.KeycloakServer)
	s.DeploymentRESTServerUrl = fmt.Sprintf("http://%s:%s", RestAddressPortForward, PortForwardRemotePort)
	s.projectID, err = auth.GetProjectID(context.TODO())
	s.NoError(err)
	s.portForwardCmd, err = portForwardToADM()
	s.client, err = restClient.NewClientWithResponses(s.DeploymentRESTServerUrl, restClient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		auth.AddRestAuthHeader(req, s.token, s.projectID)
		return nil
	}))
	s.NoError(err)
}

func killportForwardToADM(cmd *exec.Cmd) error {
	fmt.Println("kill process that port-forwards network to app-deployment-manager")
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

func portForwardToADM() (*exec.Cmd, error) {
	fmt.Println("port-forward to app-deployment-manager")

	cmd := exec.Command("kubectl", "port-forward", "-n", PortForwardServiceNamespace, PortForwardService, fmt.Sprintf("%s:%s", PortForwardLocalPort, PortForwardRemotePort), "--address", PortForwardAddress)
	err := cmd.Start()
	time.Sleep(5 * time.Second) // Give some time for port-forwarding to establish

	return cmd, err
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// TearDownTest tears down remnants of each integration test
func (s *TestSuite) TearDownTest(ctx context.Context) {
	err := killportForwardToADM(s.portForwardCmd)
	s.NoError(err)

}

/*func (s *TestSuite) TearDownSuite() {
	s.T().Log("Cleaning up deployments created during the test suite...")
	for _, displayName := range s.createdDeployments {
		s.T().Logf("Attempting to delete deployment '%s'...", displayName)
		err := deleteAndRetryUntilDeleted(s.client, displayName, retryCount, retryDelay)
		if err != nil {
			s.T().Logf("Failed to delete deployment '%s': %v", displayName, err)
		} else {
			s.T().Logf("Successfully deleted deployment '%s'", displayName)
		}
	}
}*/
