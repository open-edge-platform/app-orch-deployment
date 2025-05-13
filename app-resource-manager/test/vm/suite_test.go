// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package basic is a suite of basic functionality tests for the ADM service
package vm

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

const (
	RestAddressPortForward = "127.0.0.1"

	AdmPortForwardRemote = "8081"
	ArmPortForwardRemote = "8082"
	VMRunning            = string(armClient.VirtualMachineStatusStateSTATERUNNING)
	VMStopped            = string(armClient.VirtualMachineStatusStateSTATESTOPPED)
	dpDisplayName        = "vm-test-vm"
	vmExtDisplayName     = "virt-extension"
	dpConfigName         = "vm"
	vmExtDPConfigName    = "virt-extension"
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	ResourceRESTServerUrl string
	token                 string
	projectID             string
	deployApps            []*admClient.App
	ArmClient             *armClient.ClientWithResponses
	KeycloakServer        string
	orchDomain            string
}

// SetupTest sets up for each test
func (s *TestSuite) SetupTest() {
	autoCert, err := strconv.ParseBool(os.Getenv("AUTO_CERT"))
	s.orchDomain = os.Getenv("ORCH_DOMAIN")
	if err != nil || !autoCert || s.orchDomain == "" {
		s.orchDomain = "kind.internal"
	}
	s.KeycloakServer = fmt.Sprintf("keycloak.%s", s.orchDomain)

	s.token = token
	s.projectID = projectID
	s.ResourceRESTServerUrl = resourceRESTServerUrl
	s.ArmClient = armclient
	s.deployApps = deployApps
	s.NotEmpty(s.deployApps)
}

func TestVMSuite(t *testing.T) {
	portForwardCmd, err := utils.BringUpPortForward()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer utils.TearDownPortForward(portForwardCmd)

	autoCert, err := strconv.ParseBool(os.Getenv("AUTO_CERT"))
	orchDomain := os.Getenv("ORCH_DOMAIN")
	if err != nil || !autoCert || orchDomain == "" {
		orchDomain = "kind.internal"
	}
	keycloakServer := fmt.Sprintf("keycloak.%s", orchDomain)
	token, err = utils.SetUpAccessToken(keycloakServer, fmt.Sprintf("%s-edge-mgr", utils.SampleProject), utils.DefaultPass)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	projectID, err = utils.GetProjectID(context.TODO())
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resourceRESTServerUrl = fmt.Sprintf("http://%s:%s", RestAddressPortForward, ArmPortForwardRemote)
	armclient, err = utils.CreateArmClient(resourceRESTServerUrl, token, projectID)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	deploymentRESTServerUrl := fmt.Sprintf("http://%s:%s", RestAddressPortForward, AdmPortForwardRemote)
	admClientInstance, err := admClient.NewClientWithResponses(deploymentRESTServerUrl, admClient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		utils.AddRestAuthHeader(req, token, projectID)
		return nil
	}))
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	_, err = utils.CreateDeployment(admClientInstance, vmExtDPConfigName, vmExtDisplayName, 30)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	deployApps, err = utils.CreateDeployment(admClientInstance, dpConfigName, dpDisplayName, 10)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	suite.Run(t, new(TestSuite))

}
