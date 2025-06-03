// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
)

// TestAppWorkloadsListInvalidProjectID tests list app workload with invalid project id
func (s *TestSuite) TestAppWorkloadsListInvalidProjectID() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, s.Token, "invalidprojectid")
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(armClient, appID)
		s.Equal(retCode, 403)
		s.Error(err)
		s.Empty(appWorkloads)
		s.T().Logf("successfully handled invalid projectid to list app workloads\n")
	}
}

// TestAppWorkloadsListInvalidJWT tests list app workload with invalid JWT
func (s *TestSuite) TestAppWorkloadsListInvalidJWT() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.ProjectID)
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(armClient, appID)
		s.Equal(retCode, 401)
		s.Error(err)
		s.Empty(appWorkloads)
		s.T().Logf("successfully handled invalid JWT to list app workloads\n")
	}
}

// TestAppEndpointsListInvalidProjectID tests list app endpoints with invalid project id
func (s *TestSuite) TestAppEndpointsListInvalidProjectID() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, s.Token, "invalidprojectid")
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appEndpoints, retCode, err := utils.AppEndpointsList(armClient, appID)
		s.Equal(retCode, 403)
		s.Error(err)
		s.Empty(appEndpoints)
		s.T().Logf("successfully handled invalid projectid to list app endpoints\n")
	}
}

// TestAppEndpointsListInvalidJWT tests list app endpoints with invalid JWT
func (s *TestSuite) TestAppEndpointsListInvalidJWT() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.ProjectID)
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appEndpoints, retCode, err := utils.AppEndpointsList(armClient, appID)
		s.Equal(retCode, 401)
		s.Error(err)
		s.Empty(appEndpoints)
		s.T().Logf("successfully handled invalid JWT to list app endpoints\n")
	}
}

// TestDeletePodAuthInvalidProjectID tests delete pod with invalid project id
func (s *TestSuite) TestDeletePodAuthInvalidProjectID() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, s.Token, "invalidprojectid")
	s.NoError(err)

	retCode, err := utils.PodDelete(armClient, "namespace", "podname", "appID")
	s.Equal(retCode, 403)
	s.Error(err)
	s.T().Logf("successfully handled invalid projectid to delete pod\n")
}

// TestDeletePodAuthInvalidJWT tests delete pod with invalid jwt
func (s *TestSuite) TestDeletePodAuthInvalidJWT() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.ProjectID)
	s.NoError(err)

	retCode, err := utils.PodDelete(armClient, "namespace", "podname", "appID")
	s.Equal(retCode, 401)
	s.Error(err)
	s.T().Logf("successfully handled invalid JWT to delete pod\n")
}
