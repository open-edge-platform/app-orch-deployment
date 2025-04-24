// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/container"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

// TestVMAuthProjectIDStop tests VM stop with invalid project id
func (s *TestSuite) TestVMAuthProjectIDStop() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

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

			retCode, err = StopVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 403)
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to stop virtual machine\n")
		}
	}
}

// TestVMAuthJWTStop tests VM stop with invalid jwt
func (s *TestSuite) TestVMAuthJWTStop() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, utils.InvalidJWT, s.projectID)
	s.NoError(err)

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

			retCode, err := StopVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 401)
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to stop virtual machine\n")
		}
	}
}

// TestVMAuthProjectIDStart tests VM start with invalid project id
func (s *TestSuite) TestVMAuthProjectIDStart() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

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
			// Stop VM if running
			if currState != VMStopped {
				retCode, err := StopVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, 200)
				s.NoError(err)
				s.T().Logf("stop VM pod %s\n", appWorkload.Name)

				err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMStopped)
				s.NoError(err)
			}

			retCode, err := StartVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 403)
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to start virtual machine\n")
		}
	}
}

// TestVMAuthJWTStart tests VM start with invalid jwt
func (s *TestSuite) TestVMAuthJWTStart() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, utils.InvalidJWT, s.projectID)
	s.NoError(err)

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
			// Stop VM if not running
			if currState != VMStopped {
				retCode, err := StopVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, 200)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMStopped)
				s.NoError(err)
			}

			retCode, err := StartVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 401)
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to start virtual machine\n")
		}
	}
}

// TestVMAuthProjectIDRestart tests VM restart with invalid project id
func (s *TestSuite) TestVMAuthProjectIDRestart() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

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
				retCode, err = StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, 200)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
				s.NoError(err)
			}

			retCode, err = RestartVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 403)
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to restart virtual machine\n")
		}
	}
}

// TestVMAuthJWTRestart tests VM restart with invalid jwt
func (s *TestSuite) TestVMAuthJWTRestart() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, utils.InvalidJWT, s.projectID)
	s.NoError(err)

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
				retCode, err = StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, 200)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
				s.NoError(err)
			}

			retCode, err = RestartVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 401)
			s.Error(err)
			s.T().Logf("successfully handled invalid jwt to restart virtual machine\n")
		}
	}
}

// TestVMAuthProjectIDGetVNC tests get vnc with invalid project id
func (s *TestSuite) TestVMAuthProjectIDGetVNC() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

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
			retCode, err = GetVNC(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 403)
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to get vnc\n")
		}
	}
}

// TestVMAuthJWTGetVNC tests get vnc with invalid jwt
func (s *TestSuite) TestVMAuthJWTGetVNC() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, utils.InvalidJWT, s.projectID)
	s.NoError(err)

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
			retCode, err = GetVNC(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 401)
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to get VNC\n")
		}
	}
}
