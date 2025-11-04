// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package basic is a suite of basic functionality tests for the ADM service
package vm

import (
	"context"
	"fmt"
	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/loader"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/portforwarding"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"

	"os/exec"
	"testing"
	"time"

	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/stretchr/testify/suite"
)

const (
	VMRunning = string(armClient.ResourceV2VirtualMachineStatusStateSTATERUNNING)
	VMStopped = string(armClient.ResourceV2VirtualMachineStatusStateSTATESTOPPED)
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
	OrchDomain            string
	PortForwardCmd        map[string]*exec.Cmd
}

// SetupTest can be used for per-test setup if needed
func (s *TestSuite) SetupTest() {
	// Leave empty or add per-test setup logic here
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

	err = loader.UploadCirrosVM()
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}

	// Start the virtualization extension deployment
	virtDeploymentRequest := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.VirtualizationExtensionAppName,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "VirtExtDep",
	}
	_, _, err = deploymentutils.StartDeployment(virtDeploymentRequest)
	if err != nil {
		s.T().Fatalf("error: %v", err)

	}

	// Create a deployment for the cirros app
	cirrosDeploymentRequest := deploymentutils.StartDeploymentRequest{
		AdmClient:         s.AdmClient,
		DpPackageName:     deploymentutils.CirrosAppName,
		DeploymentType:    deploymentutils.DeploymentTypeTargeted,
		DeploymentTimeout: deploymentutils.DeploymentTimeout,
		DeleteTimeout:     deploymentutils.DeleteTimeout,
		TestName:          "CirrosDeployment",
	}

	deployID, _, err := deploymentutils.StartDeployment(cirrosDeploymentRequest)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}
	s.DeployApps, err = deploymentutils.GetDeployApps(s.AdmClient, deployID)
	if err != nil {
		s.T().Fatalf("error: %v", err)
	}
	s.NotEmpty(s.DeployApps)
}

// TearDownSuite cleans up after the entire test suite
func (s *TestSuite) TearDownSuite() {
	err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, deploymentutils.CirrosAppName, 10, 10*time.Second)
	s.NoError(err)
	err = deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, deploymentutils.VirtualizationExtensionAppName, 10, 10*time.Second)
	s.NoError(err)
	portforwarding.TearDownPortForward(s.PortForwardCmd)

}

func TestVMSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
