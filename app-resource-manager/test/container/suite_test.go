// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"

	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"github.com/stretchr/testify/suite"
)

var (
	deployApps            []*admClient.App
	token                 string
	projectID             string
	resourceRESTServerUrl string
	armclient             *armClient.ClientWithResponses
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	ResourceRESTServerUrl string
	token                 string
	projectID             string
	deployApps            []*admClient.App
	armClient             *armClient.ClientWithResponses
	admClient             *admClient.ClientWithResponses
	KeycloakServer        string
	orchDomain            string
	portForwardCmd        map[string]*exec.Cmd
}

// SetupSuite sets up the test suite once before all tests
func (s *TestSuite) SetupSuite() {
	autoCert, err := strconv.ParseBool(os.Getenv("AUTO_CERT"))
	s.orchDomain = os.Getenv("ORCH_DOMAIN")
	if err != nil || !autoCert || s.orchDomain == "" {
		s.orchDomain = "kind.internal"
	}
	s.KeycloakServer = fmt.Sprintf("keycloak.%s", s.orchDomain)

	s.token, err = utils.SetUpAccessToken(s.KeycloakServer, fmt.Sprintf("%s-edge-mgr", utils.SampleProject), utils.DefaultPass)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.projectID, err = utils.GetProjectID(context.TODO())
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}
	s.portForwardCmd, err = utils.StartPortForwarding()
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.ResourceRESTServerUrl = fmt.Sprintf("http://%s:%s", utils.RestAddressPortForward, utils.ArmPortForwardRemote)
	s.armClient, err = utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, s.projectID)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	deploymentRESTServerUrl := fmt.Sprintf("http://%s:%s", utils.RestAddressPortForward, utils.AdmPortForwardRemote)
	s.admClient, err = admClient.NewClientWithResponses(deploymentRESTServerUrl, admClient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		utils.AddRestAuthHeader(req, s.token, s.projectID)
		return nil
	}))
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.deployApps, err = utils.CreateDeployment(s.admClient, utils.NginxAppName, utils.NginxAppName, 10)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.NotEmpty(s.deployApps)
}

// SetupTest can be used for per-test setup if needed
func (s *TestSuite) SetupTest() {
	// Leave empty or add per-test setup logic here
}

// TearDownSuite cleans up after the entire test suite
func (s *TestSuite) TearDownSuite() {
	utils.TearDownPortForward(s.portForwardCmd)
}

func TestContainerSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
