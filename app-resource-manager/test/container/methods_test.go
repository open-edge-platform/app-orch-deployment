// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"net/http"
)

var appWorkloadsMethods = map[string]int{
	http.MethodPut:    405,
	http.MethodGet:    200,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPost:   405,
}

var appEndpointsMethods = map[string]int{
	http.MethodPut:    405,
	http.MethodGet:    200,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPost:   405,
}

var deleteMethods = map[string]int{
	http.MethodPut:    200,
	http.MethodGet:    405,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPost:   405,
}

// TestListMethods tests both app workload and endpoint service with different HTTP methods
func (s *TestSuite) TestListMethods() {
	for _, app := range s.deployApps {
		appID := *app.Id

		for method, expectedStatus := range appWorkloadsMethods {
			res, err := MethodAppWorkloadsList(method, s.ResourceRESTServerUrl, appID, s.token, s.projectID)
			s.NoError(err)
			s.Equal(expectedStatus, res.StatusCode)
			s.T().Logf("list app workloads method: %s (%d)\n", method, res.StatusCode)
		}

		for method, expectedStatus := range appEndpointsMethods {
			res, err := MethodAppEndpointsList(method, s.ResourceRESTServerUrl, appID, s.token, s.projectID)
			s.NoError(err)
			s.Equal(expectedStatus, res.StatusCode)
			s.T().Logf("list app endpoints method: %s (%d)\n", method, res.StatusCode)
		}
	}
}

// TestDeletePodMethod tests delete pod with different HTTP methods
func (s *TestSuite) TestDeletePodMethod() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		if len(*appWorkloads) != 1 {
			s.T().Errorf("invalid app workloads len: %+v expected len 1\n", len(*appWorkloads))
		}

		s.T().Logf("app workloads len: %+v\n", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			namespace := *appWorkload.Namespace
			podName := appWorkload.Name

			for method, expectedStatus := range deleteMethods {
				if expectedStatus == 200 {
					err = GetPodStatus(s.ArmClient, appID, appWorkload.Id.String(), "STATE_RUNNING")
					s.NoError(err)
				}

				res, err := MethodPodDelete(method, s.ResourceRESTServerUrl, namespace, podName, s.token, s.projectID)
				s.NoError(err)
				s.Equal(expectedStatus, res.StatusCode)

				if expectedStatus == 200 {
					err = WaitPodDelete(s.ArmClient, appID)
					s.NoError(err)
				}

				s.T().Logf("delete pod method: %s (%d)\n", method, res.StatusCode)
			}
		}
	}
}
