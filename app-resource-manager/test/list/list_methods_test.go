// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"net/http"
)

func (s *TestSuite) TestListMethods() {
	for _, app := range s.deployApps {
		appId := *app.Id

		res, err := methodsListAppWorkloads(http.MethodGet, s.ResourceRESTServerUrl, appId, s.token, s.projectID)
		s.NoError(err)
		s.Equal(200, res.StatusCode)
		s.T().Logf("list app workloads method: %s code: %d\n", http.MethodGet, res.StatusCode)

		res, err = methodsListAppWorkloads(http.MethodDelete, s.ResourceRESTServerUrl, appId, s.token, s.projectID)
		s.NoError(err)
		s.Equal(405, res.StatusCode)
		s.T().Logf("list app workloads method: %s code: %d\n", http.MethodDelete, res.StatusCode)

		res, err = methodsListAppWorkloads(http.MethodPut, s.ResourceRESTServerUrl, appId, s.token, s.projectID)
		s.NoError(err)
		s.Equal(405, res.StatusCode)
		s.T().Logf("list app workloads method: %s code: %d\n", http.MethodPut, res.StatusCode)

		res, err = methodsListAppWorkloads(http.MethodPatch, s.ResourceRESTServerUrl, appId, s.token, s.projectID)
		s.NoError(err)
		s.Equal(405, res.StatusCode)
		s.T().Logf("list app workloads method: %s code: %d\n", http.MethodPatch, res.StatusCode)

		res, err = methodsListAppEndpoints(http.MethodGet, s.ResourceRESTServerUrl, appId, s.token, s.projectID)
		s.NoError(err)
		s.Equal(200, res.StatusCode)
		s.T().Logf("list app endpoints method: %s code: %d\n", http.MethodGet, res.StatusCode)

		res, err = methodsListAppEndpoints(http.MethodDelete, s.ResourceRESTServerUrl, appId, s.token, s.projectID)
		s.NoError(err)
		s.Equal(405, res.StatusCode)
		s.T().Logf("list app endpoints method: %s code: %d\n", http.MethodDelete, res.StatusCode)

		res, err = methodsListAppEndpoints(http.MethodPut, s.ResourceRESTServerUrl, appId, s.token, s.projectID)
		s.NoError(err)
		s.Equal(405, res.StatusCode)
		s.T().Logf("list app endpoints method: %s code: %d\n", http.MethodPut, res.StatusCode)

		res, err = methodsListAppEndpoints(http.MethodPatch, s.ResourceRESTServerUrl, appId, s.token, s.projectID)
		s.NoError(err)
		s.Equal(405, res.StatusCode)
		s.T().Logf("list app endpoints method: %s code: %d\n", http.MethodPatch, res.StatusCode)
	}
}
