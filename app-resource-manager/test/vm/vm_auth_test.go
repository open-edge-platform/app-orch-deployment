// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/auth"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/list"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

// TestAuthProjectIDStopVM tests stop vm with invalid project id
func (s *TestSuite) TestAuthProjectIDStopVM() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

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

			err = StopVirtualMachine(armClient, appID, appWorkload.Id.String())
			s.Equal(err.Error(), "failed to stop virtual machine: <nil>, status: 403")
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to stop virtual machine\n")
		}
	}
}

// TestAuthJWTStopVM tests stop vm with invalid jwt
func (s *TestSuite) TestAuthJWTStopVM() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.projectID)
	s.NoError(err)

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

			err = StopVirtualMachine(armClient, appID, appWorkload.Id.String())
			s.Equal(err.Error(), "failed to stop virtual machine: <nil>, status: 401")
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to stop virtual machine\n")
		}
	}
}

// TestAuthProjectIDStartVM tests start vm with invalid project id
func (s *TestSuite) TestAuthProjectIDStartVM() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

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
			// Stop VM if running
			if currState != VMStopped {
				err = StopVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
				s.NoError(err)
				s.T().Logf("stop VM pod %s\n", appWorkload.Name)

				err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMStopped)
				s.NoError(err)
			}

			err = StartVirtualMachine(armClient, appID, appWorkload.Id.String())
			s.Equal(err.Error(), "failed to start virtual machine: <nil>, status: 403")
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to start virtual machine\n")
		}
	}
}

// TestAuthJWTStartVM tests start vm with invalid jwt
func (s *TestSuite) TestAuthJWTStartVM() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.projectID)
	s.NoError(err)

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
			// Stop VM if not running
			if currState != VMStopped {
				err = StopVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMStopped)
				s.NoError(err)
			}

			err = StartVirtualMachine(armClient, appID, appWorkload.Id.String())
			s.Equal(err.Error(), "failed to start virtual machine: <nil>, status: 401")
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to start virtual machine\n")
		}
	}
}

// TestAuthProjectIDRestartVM tests restart vm with invalid project id
func (s *TestSuite) TestAuthProjectIDRestartVM() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

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

			err = RestartVirtualMachine(armClient, appID, appWorkload.Id.String())
			s.Equal(err.Error(), "failed to restart virtual machine: <nil>, status: 403")
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to restart virtual machine\n")
		}
	}
}

// TestAuthJWTRestartVM tests restart vm with invalid jwt
func (s *TestSuite) TestAuthJWTRestartVM() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.projectID)
	s.NoError(err)

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

			err = RestartVirtualMachine(armClient, appID, appWorkload.Id.String())
			s.Equal(err.Error(), "failed to restart virtual machine: <nil>, status: 401")
			s.Error(err)
			s.T().Logf("successfully handled invalid jwt to restart virtual machine\n")
		}
	}
}

// TestAuthProjectIDGetVNC tests get vnc with invalid project id
func (s *TestSuite) TestAuthProjectIDGetVNC() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

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
			err = GetVNC(armClient, appID, appWorkload.Id.String())
			s.Equal(err.Error(), "failed to get VNC: <nil>, status: 403")
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to get vnc\n")
		}
	}
}

// TestAuthJWTGetVNC tests get vnc with invalid jwt
func (s *TestSuite) TestAuthJWTGetVNC() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.projectID)
	s.NoError(err)

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
			err = GetVNC(armClient, appID, appWorkload.Id.String())
			s.Equal(err.Error(), "failed to get VNC: <nil>, status: 401")
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to get VNC\n")
		}
	}
}
