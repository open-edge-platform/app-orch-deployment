// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/list"
)

var methods = map[string]int{
	http.MethodPut:    200,
	http.MethodGet:    405,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPost:   405,
}

func (s *TestSuite) TestDeleteMethods() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, err := list.AppWorkloadsList(s.ArmClient, appID)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		if len(*appWorkloads) != 1 {
			s.T().Errorf("invalid app workloads len: %+v expected len 1\n", len(*appWorkloads))
		}

		s.T().Logf("app Workloads len: %+v\n", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			namespace := *appWorkload.Namespace
			podName := appWorkload.Name

			for method, expectedStatus := range methods {
				if expectedStatus == 200 {
					err = GetPodStatus(s.ArmClient, appID, appWorkload.Id.String(), "STATE_RUNNING")
					s.NoError(err)
				}

				res, err := MethodPodDelete(method, s.ResourceRESTServerUrl, namespace, podName, s.token, s.projectID)
				s.NoError(err)
				s.Equal(expectedStatus, res.StatusCode)

				s.T().Logf("delete pod method: %s (%d)\n", method, res.StatusCode)
			}
		}
	}
}
