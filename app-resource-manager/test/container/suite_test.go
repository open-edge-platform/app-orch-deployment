// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"net/http"

	"testing"

	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/deploy"
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
	KeycloakServer         = "keycloak.kind.internal"

	AdmPortForwardRemote = "8081"
	ArmPortForwardRemote = "8082"
	dpDisplayName        = "nginx-test-container"
	dpConfigName         = "nginx"
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
	s.ResourceRESTServerUrl = resourceRESTServerUrl
	s.ArmClient = armclient
	s.deployApps = deployApps
	s.NotEmpty(s.deployApps)
}

func TestContainerSuite(t *testing.T) {
	portForwardCmd, err := utils.BringUpPortForward()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer utils.TearDownPortForward(portForwardCmd)

	token, err = utils.SetUpAccessToken(KeycloakServer)
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

	deployApps, err = deploy.CreateDeployment(admClientInstance, dpConfigName, dpDisplayName, 10)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	suite.Run(t, new(TestSuite))
}
