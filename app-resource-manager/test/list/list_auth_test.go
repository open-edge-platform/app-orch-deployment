// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/auth"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

// TestList tests list endpoints
func (s *TestSuite) TestAuthProjectIDList() {
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

func (s *TestSuite) TestAuthJWTList() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.projectID)
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
