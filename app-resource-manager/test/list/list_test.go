// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package list

// TestList tests list endpoints
func (s *TestSuite) TestList() {
	for _, app := range s.deployApps {
		appId := *app.Id
		appWorkloads, err := ListAppWorkloads(s.ArmClient, appId)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// nginx app workload len should be 1
		if len(*appWorkloads) != 1 {
			s.T().Errorf("invalid app workloads len: %+v expected len 1\n", len(*appWorkloads))
		}

		s.T().Logf("app Workloads len: %+v\n", len(*appWorkloads))

		appEndpoints, err := ListAppEndpoints(s.ArmClient, appId)
		s.NoError(err)
		s.NotEmpty(appEndpoints)
	}
}
