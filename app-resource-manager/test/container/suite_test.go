// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	ResourceRESTServerUrl string
	Token                 string
	ProjectID             string
	DeployApps            []*admClient.App
	ArmClient             *armClient.ClientWithResponses
	AdmClient             *admClient.ClientWithResponses
	KeycloakServer        string
	OrchDomain            string
	PortForwardCmd        map[string]*exec.Cmd
}

// SetupSuite sets up the test suite once before all tests
func (s *TestSuite) SetupSuite() {
	autoCert, err := strconv.ParseBool(os.Getenv("AUTO_CERT"))
	s.OrchDomain = os.Getenv("ORCH_DOMAIN")
	if err != nil || !autoCert || s.OrchDomain == "" {
		s.OrchDomain = "kind.internal"
	}
	s.KeycloakServer = fmt.Sprintf("keycloak.%s", s.OrchDomain)

	s.Token, err = utils.SetUpAccessToken(s.KeycloakServer, fmt.Sprintf("%s-edge-mgr", utils.SampleProject), utils.DefaultPass)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.ProjectID, err = utils.GetProjectID(context.TODO())
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}
	s.PortForwardCmd, err = utils.StartPortForwarding()
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.ResourceRESTServerUrl = fmt.Sprintf("http://%s:%s", utils.RestAddressPortForward, utils.ArmPortForwardRemote)
	s.ArmClient, err = utils.CreateArmClient(s.ResourceRESTServerUrl, s.Token, s.ProjectID)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	deploymentRESTServerUrl := fmt.Sprintf("http://%s:%s", utils.RestAddressPortForward, utils.AdmPortForwardRemote)
	s.AdmClient, err = admClient.NewClientWithResponses(deploymentRESTServerUrl, admClient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		utils.AddRestAuthHeader(req, s.Token, s.ProjectID)
		return nil
	}))
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	// Create a deployment for the cirros app
	nginxDeploymentRequest := utils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     utils.NginxAppName,
		DeploymentType:    utils.DeploymentTypeTargeted,
		DeploymentTimeout: utils.DeploymentTimeout,
		DeleteTimeout:     utils.DeleteTimeout,
		TestName:          "NginxDeployment",
	}

	deployID, _, err := utils.StartDeployment(nginxDeploymentRequest)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}
	s.DeployApps, err = utils.GetDeployApps(s.AdmClient, deployID)
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
	err := utils.DeleteAndRetryUntilDeleted(s.AdmClient, utils.NginxAppName, 10, 10*time.Second)
	s.NoError(err)
	utils.TearDownPortForward(s.PortForwardCmd)
}

func TestContainerTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
