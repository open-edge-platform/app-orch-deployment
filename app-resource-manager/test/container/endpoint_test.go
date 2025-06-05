// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"net/http"
)

// TestListWorkloads tests listing app workloads
func (s *TestSuite) TestListWorkloads() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))
		s.T().Logf("app workloads len: %+v\n", len(*appWorkloads))
	}
}

// TestListEndpoints tests listing app endpoints
func (s *TestSuite) TestListEndpoints() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appEndpoints, retCode, err := utils.AppEndpointsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appEndpoints)
	}
}

// TestDeletePod tests delete pod endpoint
func (s *TestSuite) TestDeletePod() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			retCode, err := utils.PodDelete(s.ArmClient, *appWorkload.Namespace, appWorkload.Name, appID)
			s.Equal(retCode, http.StatusOK)
			s.NoError(err)

			s.T().Logf("deleted pod %s\n", appWorkload.Name)
		}
	}
}
