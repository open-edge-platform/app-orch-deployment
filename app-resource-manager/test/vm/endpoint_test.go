// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/container"
)

// TestGetVNC tests get vnc endpoint
func (s *TestSuite) TestGetVNC() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			retCode, err = GetVNC(s.ArmClient, appID, appWorkload.Id)
			s.Equal(retCode, 200)
			s.NoError(err)
			s.T().Logf("get VM pod %s\n", appWorkload.Name)
		}
	}
}

// TestVMStart tests VM start endpoint
func (s *TestSuite) TestVMStart() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
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
				retCode, err = StopVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, 200)
				s.NoError(err)
				s.T().Logf("stop VM pod %s\n", appWorkload.Name)

				err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMStopped)
				s.NoError(err)
			}

			retCode, err = StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
			s.Equal(retCode, 200)
			s.NoError(err)
			s.T().Logf("start VM pod %s\n", appWorkload.Name)

			err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
			s.NoError(err)

		}
	}
}

// TestVMStop tests VM stop endpoint
func (s *TestSuite) TestVMStop() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
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
				retCode, err := StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, 200)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
				s.NoError(err)
			}

			retCode, err := StopVirtualMachine(s.ArmClient, appID, appWorkload.Id)
			s.Equal(retCode, 200)
			s.NoError(err)
			s.T().Logf("stop VM pod %s\n", appWorkload.Name)

			err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMStopped)
			s.NoError(err)
		}
	}
}

// TestVMRestart tests VM restart endpoint
func (s *TestSuite) TestVMRestart() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
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
				retCode, err := StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, 200)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
				s.NoError(err)
			}

			retCode, err := RestartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
			s.Equal(retCode, 200)
			s.NoError(err)
			s.T().Logf("restart VM pod %s\n", appWorkload.Name)

			err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, currState)
			s.NoError(err)
		}
	}
}
