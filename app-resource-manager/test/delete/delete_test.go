// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/list"
)

func (s *TestSuite) TestDelete() {
	for _, app := range s.deployApps {
		appId := *app.Id
		appWorkloads, err := list.ListAppWorkloads(s.ArmClient, appId)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// nginx app workload len should be 1
		if len(*appWorkloads) != 1 {
			s.T().Errorf("invalid app workloads len: %+v expected len 1\n", len(*appWorkloads))
		}

		s.T().Logf("app Workloads len: %+v\n", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			s.T().Logf("app Workload appWorkload.Namespace: %+v\n", *appWorkload.Namespace)
			s.T().Logf("app Workload appWorkload.Name: %+v\n", appWorkload.Name)

			err = DeletePod(s.ArmClient, appId, appWorkload.Name, *appWorkload.Namespace)
			s.NoError(err)
			s.T().Logf("deleted pod %s\n", appWorkload.Name)
		}
		time.Sleep(5 * time.Second) // Wait for deletion
	}
}
