// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package basic is a suite of basic functionality tests for the ADM service
package vm

import (
	"context"
	"fmt"
	"net/http"

	"testing"

	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/auth"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"github.com/stretchr/testify/suite"
)

var (
	deployApps []*admClient.App
	token      string
	projectID  string
	armclient  *armClient.ClientWithResponses
)

const (
	RestAddressPortForward = "127.0.0.1"
	KeycloakServer         = "keycloak.kind.internal"

	AdmPortForwardRemote = "8081"
	ArmPortForwardRemote = "8082"
	VMRunning            = "STATE_RUNNING"
	VMStopped            = "STATE_STOPPED"
	dpDisplayName        = "vm-test-vm"
	vmExtDisplayName     = "virt-extension"
	dpName               = "vm"
	vmExtDPName          = "virt-extension"
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	ResourceRESTServerUrl string
	token                 string
	projectID             string
	deployApps            []*admClient.App
	ArmClient             *armClient.ClientWithResponses
}

// SetupTest sets up for each test
func (s *TestSuite) SetupTest() {
	s.token = token
	s.projectID = projectID

	s.ResourceRESTServerUrl = fmt.Sprintf("http://%s:%s", RestAddressPortForward, ArmPortForwardRemote)

	s.ArmClient = armclient
	s.deployApps = deployApps
	s.NotEmpty(s.deployApps)
}

func TestTestSuite(t *testing.T) {
	portForwardCmd, err := utils.BringUpPortForward()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer utils.TearDownPortForward(portForwardCmd)

	token, err = auth.SetUpAccessToken(KeycloakServer)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	projectID, err = auth.GetProjectID(context.TODO())
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resourceRESTServerUrl := fmt.Sprintf("http://%s:%s", RestAddressPortForward, ArmPortForwardRemote)
	armclient, err = utils.CreateArmClient(resourceRESTServerUrl, token, projectID)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	deploymentRESTServerUrl := fmt.Sprintf("http://%s:%s", RestAddressPortForward, AdmPortForwardRemote)
	admClientInstance, err := admClient.NewClientWithResponses(deploymentRESTServerUrl, admClient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		auth.AddRestAuthHeader(req, token, projectID)
		return nil
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// todo deploy 1. base extension (this should already be deployed when cluster creates 2. virt dp 3. vm app
	_, err = deploy.CreateDeployment(admClientInstance, vmExtDPName, vmExtDisplayName, 30)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	deployApps, err = deploy.CreateDeployment(admClientInstance, dpName, dpDisplayName, 10)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	suite.Run(t, new(TestSuite))

}
