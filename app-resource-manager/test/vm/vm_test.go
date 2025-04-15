// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/list"
)

// TestList tests VM endpoints
func (s *TestSuite) TestVM() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, err := list.AppWorkloadsList(s.ArmClient, appID)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		if len(*appWorkloads) != 1 {
			s.T().Errorf("invalid app workloads len: %+v expected len 1\n", len(*appWorkloads))
		}

		for _, appWorkload := range *appWorkloads {
			err = GetVNC(s.ArmClient, appID, appWorkload.Id.String())
			s.NoError(err)
			s.T().Logf("get VM pod %s\n", appWorkload.Name)
		}
	}
}

// TestStartVM tests start VM endpoint
func (s *TestSuite) TestStartVM() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, err := list.AppWorkloadsList(s.ArmClient, appID)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		if len(*appWorkloads) != 1 {
			s.T().Errorf("invalid app workloads len: %+v expected len 1\n", len(*appWorkloads))
		}

		for _, appWorkload := range *appWorkloads {
			// Stop VM if running
			currState := string(*appWorkload.VirtualMachine.Status.State)
			if currState != VMStopped {
				err = StopVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
				s.NoError(err)
				s.T().Logf("stop VM pod %s\n", appWorkload.Name)

				err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMStopped)
				s.NoError(err)
			}

			// Start
			err = StartVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
			s.NoError(err)
			s.T().Logf("start VM pod %s\n", appWorkload.Name)

			err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMRunning)
			s.NoError(err)

		}
	}
}

// TestStopVM tests stop VM endpoint
func (s *TestSuite) TestStopVM() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, err := list.AppWorkloadsList(s.ArmClient, appID)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		if len(*appWorkloads) != 1 {
			s.T().Errorf("invalid app workloads len: %+v expected len 1\n", len(*appWorkloads))
		}

		for _, appWorkload := range *appWorkloads {
			// Start VM if not running
			currState := string(*appWorkload.VirtualMachine.Status.State)
			if currState != VMRunning {
				err = StartVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMRunning)
				s.NoError(err)
			}

			// Stop
			err = StopVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
			s.NoError(err)
			s.T().Logf("stop VM pod %s\n", appWorkload.Name)

			err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMStopped)
			s.NoError(err)
		}
	}
}

// TestRestartVM tests restart VM endpoint
func (s *TestSuite) TestRestartVM() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, err := list.AppWorkloadsList(s.ArmClient, appID)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		if len(*appWorkloads) != 1 {
			s.T().Errorf("invalid app workloads len: %+v expected len 1\n", len(*appWorkloads))
		}

		for _, appWorkload := range *appWorkloads {
			currState := string(*appWorkload.VirtualMachine.Status.State)
			// Start VM if not running
			if currState != VMRunning {
				err = StartVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMRunning)
				s.NoError(err)
			}

			err = RestartVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
			s.NoError(err)
			s.T().Logf("restart VM pod %s\n", appWorkload.Name)

			err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), currState)
			s.NoError(err)
		}
	}
}
