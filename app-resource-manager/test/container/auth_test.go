// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

// TestListAuthProjectID tests list both app workload and endpoint service with invalid project id
func (s *TestSuite) TestListAuthProjectID() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, err := AppWorkloadsList(armClient, appID)
		s.Equal(err.Error(), "failed to list app workloads: <nil>, status: 403")
		s.Error(err)
		s.Empty(appWorkloads)
		s.T().Logf("successfully handled invalid projectid to list app workloads\n")

		appEndpoints, err := AppEndpointsList(armClient, appID)
		s.Equal(err.Error(), "failed to list app endpoints: <nil>, status: 403")
		s.Error(err)
		s.Empty(appEndpoints)
		s.T().Logf("successfully handled invalid projectid to list app endpoints\n")
	}
}

// TestListAuthJWT tests list both app workload and endpoint service with invalid jwt
func (s *TestSuite) TestListAuthJWT() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, utils.InvalidJWT, s.projectID)
	s.NoError(err)

	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, err := AppWorkloadsList(armClient, appID)
		s.Equal(err.Error(), "failed to list app workloads: <nil>, status: 401")
		s.Error(err)
		s.Empty(appWorkloads)
		s.T().Logf("successfully handled invalid JWT to list app workloads\n")

		appEndpoints, err := AppEndpointsList(armClient, appID)
		s.Equal(err.Error(), "failed to list app endpoints: <nil>, status: 401")
		s.Error(err)
		s.Empty(appEndpoints)
		s.T().Logf("successfully handled invalid JWT to list app endpoints\n")
	}
}

// TestDeletePodAuthProjectID tests delete pod with invalid project id
func (s *TestSuite) TestDeletePodAuthProjectID() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

	err = PodDelete(armClient, "namespace", "podname", "appID")
	s.Equal(err.Error(), "failed to delete pod: <nil>, status: 403")
	s.Error(err)
	s.T().Logf("successfully handled invalid projectid to delete pod\n")
}

// TestDeletePodAuthJWT tests delete pod with invalid jwt
func (s *TestSuite) TestDeletePodAuthJWT() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, utils.InvalidJWT, s.projectID)
	s.NoError(err)

	err = PodDelete(armClient, "namespace", "podname", "appID")
	s.Equal(err.Error(), "failed to delete pod: <nil>, status: 401")
	s.Error(err)
	s.T().Logf("successfully handled invalid JWT to delete pod\n")
}
