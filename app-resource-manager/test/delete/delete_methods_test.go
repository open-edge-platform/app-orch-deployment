// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/list"
)

func (s *TestSuite) TestDeleteMethods() {
	for _, app := range s.deployApps {
		appId := *app.Id
		appWorkloads, err := list.ListAppWorkloads(s.ArmClient, appId)
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

			res, err := methodsPodDelete(http.MethodPut, s.ResourceRESTServerUrl, namespace, podName, s.token, s.projectID)
			s.NoError(err)
			s.Equal(200, res.StatusCode)
			s.T().Logf("delete pod method: %s (%d)\n", http.MethodPut, res.StatusCode)

			res, err = methodsPodDelete(http.MethodGet, s.ResourceRESTServerUrl, namespace, podName, s.token, s.projectID)
			s.NoError(err)
			s.Equal(405, res.StatusCode)
			s.T().Logf("delete pod method: %s (%d)\n", http.MethodGet, res.StatusCode)

			res, err = methodsPodDelete(http.MethodDelete, s.ResourceRESTServerUrl, namespace, podName, s.token, s.projectID)
			s.NoError(err)
			s.Equal(405, res.StatusCode)
			s.T().Logf("delete pod method: %s (%d)\n", http.MethodDelete, res.StatusCode)

			res, err = methodsPodDelete(http.MethodPatch, s.ResourceRESTServerUrl, namespace, podName, s.token, s.projectID)
			s.NoError(err)
			s.Equal(405, res.StatusCode)
			s.T().Logf("delete pod method: %s (%d)\n", http.MethodPatch, res.StatusCode)
		}
	}
}
