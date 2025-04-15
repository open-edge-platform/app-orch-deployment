// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package list

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

// TestListMethods tests list methods
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
