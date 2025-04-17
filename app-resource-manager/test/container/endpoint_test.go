// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

// TestList tests both app workload and service endpoints
func (s *TestSuite) TestList() {
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

		appEndpoints, retCode, err := AppEndpointsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appEndpoints)
	}
}

// TestDeletePod tests delete pod endpoint
func (s *TestSuite) TestDeletePod() {
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

		for _, appWorkload := range *appWorkloads {
			retCode, err := PodDelete(s.ArmClient, *appWorkload.Namespace, appWorkload.Name, appID)
			s.Equal(retCode, 200)
			s.NoError(err)

			s.T().Logf("deleted pod %s\n", appWorkload.Name)
		}
	}
}
