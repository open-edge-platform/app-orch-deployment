// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"context"
	"fmt"
	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"os/exec"
	"strconv"
)

type BaseSuite struct {
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
func (s *BaseSuite) SetupSuite() {
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

	err = utils.UploadCirrosVM()
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	_, err = utils.CreateDeployment(s.AdmClient, utils.VirtualizationExtensionAppName, utils.VirtualizationExtensionAppName, 30)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.DeployApps, err = utils.CreateDeployment(s.AdmClient, utils.CirrosAppName, utils.CirrosAppName, 10)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	s.NotEmpty(s.DeployApps)
}

// SetupTest can be used for per-test setup if needed
func (s *BaseSuite) SetupTest() {
	// Leave empty or add per-test setup logic here
}
