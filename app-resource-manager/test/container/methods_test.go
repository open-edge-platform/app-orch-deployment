// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

var appWorkloadsMethods = map[string]int{
	http.MethodPut:    http.StatusMethodNotAllowed,
	http.MethodGet:    http.StatusOK,
	http.MethodDelete: http.StatusMethodNotAllowed,
	http.MethodPatch:  http.StatusMethodNotAllowed,
	http.MethodPost:   http.StatusMethodNotAllowed,
}

var appEndpointsMethods = map[string]int{
	http.MethodPut:    http.StatusMethodNotAllowed,
	http.MethodGet:    http.StatusOK,
	http.MethodDelete: http.StatusMethodNotAllowed,
	http.MethodPatch:  http.StatusMethodNotAllowed,
	http.MethodPost:   http.StatusMethodNotAllowed,
}

var deleteMethods = map[string]int{
	http.MethodPut:    http.StatusOK,
	http.MethodGet:    http.StatusMethodNotAllowed,
	http.MethodDelete: http.StatusMethodNotAllowed,
	http.MethodPatch:  http.StatusMethodNotAllowed,
	http.MethodPost:   http.StatusMethodNotAllowed,
}

// TestListMethods tests both app workload and endpoint service with different HTTP methods
func (s *TestSuite) TestListMethods() {
	for _, app := range s.DeployApps {
		appID := *app.Id

		for method, expectedStatus := range appWorkloadsMethods {
			res, err := utils.MethodAppWorkloadsList(method, s.ResourceRESTServerUrl, appID, s.Token, s.ProjectID)
			s.NoError(err)
			s.Equal(expectedStatus, res.StatusCode)
			s.T().Logf("list app workloads method: %s (%d)\n", method, res.StatusCode)
		}

		for method, expectedStatus := range appEndpointsMethods {
			res, err := utils.MethodAppEndpointsList(method, s.ResourceRESTServerUrl, appID, s.Token, s.ProjectID)
			s.NoError(err)
			s.Equal(expectedStatus, res.StatusCode)
			s.T().Logf("list app endpoints method: %s (%d)\n", method, res.StatusCode)
		}
	}
}

// TestDeletePodMethod tests delete pod with different HTTP methods
func (s *TestSuite) TestDeletePodMethod() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		s.T().Logf("app workloads len: %+v\n", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			namespace := *appWorkload.Namespace
			podName := appWorkload.Name

			for method, expectedStatus := range deleteMethods {
				if expectedStatus == http.StatusOK {
					err = utils.GetPodStatus(s.ArmClient, appID, appWorkload.Id, "STATE_RUNNING")
					s.NoError(err)
				}

				res, err := utils.MethodPodDelete(method, s.ResourceRESTServerUrl, namespace, podName, s.Token, s.ProjectID)
				s.NoError(err)
				s.Equal(expectedStatus, res.StatusCode)

				if expectedStatus == http.StatusOK {
					err = utils.WaitPodDelete(s.ArmClient, appID)
					s.NoError(err)
				}

				s.T().Logf("delete pod method: %s (%d)\n", method, res.StatusCode)
			}
		}
	}
}
