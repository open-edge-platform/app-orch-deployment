// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/portforwarding"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
	"github.com/stretchr/testify/suite"
	"os/exec"
	"testing"

	"time"
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	ResourceRESTServerUrl string
	Token                 string
	ProjectID             string
	DeployApps            []*admClient.DeploymentV1App
	ArmClient             *armClient.ClientWithResponses
	AdmClient             *admClient.ClientWithResponses
	KeycloakServer        string
	OrchDomain            string
	PortForwardCmd        map[string]*exec.Cmd
}

// SetupSuite sets up the test suite once before all tests
func (s *TestSuite) SetupSuite() {
	var err error
	s.Token, err = auth.SetUpAccessToken(auth.GetKeycloakServer())
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.ProjectID, err = auth.GetProjectID(context.TODO())
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}
	s.PortForwardCmd, err = portforwarding.StartPortForwarding()
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.ResourceRESTServerUrl = fmt.Sprintf("http://%s:%s", types.RestAddressPortForward, types.ArmPortForwardRemote)
	s.ArmClient, err = clients.CreateArmClient(s.ResourceRESTServerUrl, s.Token, s.ProjectID)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	deploymentRESTServerUrl := fmt.Sprintf("http://%s:%s", types.RestAddressPortForward, types.AdmPortForwardRemote)
	s.AdmClient, err = clients.CreateAdmClient(deploymentRESTServerUrl, s.Token, s.ProjectID)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	// Create a deployment for the nginx app
	nginxDeploymentRequest := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.NginxAppName,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "NginxDeployment",
	}

	deployID, _, err := deploymentutils.StartDeployment(nginxDeploymentRequest)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}
	s.DeployApps, err = deploymentutils.GetDeployApps(s.AdmClient, deployID)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.NotEmpty(s.DeployApps)
}

// SetupTest can be used for per-test setup if needed
func (s *TestSuite) SetupTest() {
	// Leave empty or add per-test setup logic here
}

// TearDownSuite cleans up after the entire test suite
func (s *TestSuite) TearDownSuite() {
	err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, deploymentutils.NginxAppName, 10, 10*time.Second)
	s.NoError(err)
	portforwarding.TearDownPortForward(s.PortForwardCmd)
}

func TestContainerTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
