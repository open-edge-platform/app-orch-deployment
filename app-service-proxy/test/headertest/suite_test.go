// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package headertest

import (
	"context"
	"fmt"

	"testing"

	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/git"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/loader"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/portforwarding"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
	"github.com/stretchr/testify/suite"
	"os"
	"os/exec"
	"path/filepath"
	"time"
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
	HarborSecret          string
	PortForwardCmd        map[string]*exec.Cmd
}

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

	httpbinPath, err := git.CloneHttpbin()
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(filepath.Dir(httpbinPath))) // Clean up the temporary directory after upload
	secret, _ := GetCliSecretHarbor("https://registry-oci.kind.internal", s.Token)
	err = loader.UploadHttpbinHelm(httpbinPath, secret)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	err = loader.UploadHttpbin()
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	httpbinDeploymentRequest := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.HttpbinAppName,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "dep",
		ReuseFlag:         false,
	}

	deployID, _, err := deploymentutils.StartDeployment(httpbinDeploymentRequest)
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
	depName := deploymentutils.HttpbinAppName + "-dep"
	err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient,
		depName, 10, 10*time.Second)
	s.NoError(err)
	portforwarding.TearDownPortForward(s.PortForwardCmd)
}

func TestHeaderTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
