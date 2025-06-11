// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/clients"
)

// TestVMAuthInvalidProjectIDStop tests VM stop with invalid project id
func (s *TestSuite) TestVMAuthInvalidProjectIDStop() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, s.Token, "invalidprojectid")
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
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
				retCode, err := utils.StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = utils.GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
				s.NoError(err)
			}

			retCode, err = utils.StopVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 403)
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to stop virtual machine\n")
		}
	}
}

// TestVMAuthInvalidJWTStop tests VM stop with invalid jwt
func (s *TestSuite) TestVMAuthInvalidJWTStop() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.ProjectID)
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
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
				retCode, err := utils.StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = utils.GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
				s.NoError(err)
			}

			retCode, err := utils.StopVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 401)
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to stop virtual machine\n")
		}
	}
}

// TestVMAuthInvalidProjectIDStart tests VM start with invalid project id
func (s *TestSuite) TestVMAuthInvalidProjectIDStart() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, s.Token, "invalidprojectid")
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
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
				retCode, err := utils.StopVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				s.T().Logf("stop VM pod %s\n", appWorkload.Name)

				err = utils.GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMStopped)
				s.NoError(err)
			}

			retCode, err := utils.StartVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 403)
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to start virtual machine\n")
		}
	}
}

// TestVMAuthInvalidJWTStart tests VM start with invalid jwt
func (s *TestSuite) TestVMAuthInvalidJWTStart() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.ProjectID)
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
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
				retCode, err := utils.StopVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = utils.GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMStopped)
				s.NoError(err)
			}

			retCode, err := utils.StartVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 401)
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to start virtual machine\n")
		}
	}
}

// TestVMAuthInvalidProjectIDRestart tests VM restart with invalid project id
func (s *TestSuite) TestVMAuthInvalidProjectIDRestart() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, s.Token, "invalidprojectid")
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			currState := string(*appWorkload.VirtualMachine.Status.State)
			// Start VM if not running
			if currState != VMRunning {
				retCode, err = utils.StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = utils.GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
				s.NoError(err)
			}

			retCode, err = utils.RestartVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 403)
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to restart virtual machine\n")
		}
	}
}

// TestVMAuthInvalidJWTRestart tests VM restart with invalid jwt
func (s *TestSuite) TestVMAuthInvalidJWTRestart() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.ProjectID)
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			currState := string(*appWorkload.VirtualMachine.Status.State)
			// Start VM if not running
			if currState != VMRunning {
				retCode, err = utils.StartVirtualMachine(s.ArmClient, appID, appWorkload.Id)
				s.Equal(retCode, http.StatusOK)
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = utils.GetVMStatus(s.ArmClient, appID, appWorkload.Id, VMRunning)
				s.NoError(err)
			}

			retCode, err = utils.RestartVirtualMachine(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 401)
			s.Error(err)
			s.T().Logf("successfully handled invalid jwt to restart virtual machine\n")
		}
	}
}

// TestVMAuthInvalidProjectIDGetVNC tests get vnc with invalid project id
func (s *TestSuite) TestVMAuthInvalidProjectIDGetVNC() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, s.Token, "invalidprojectid")
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			retCode, err = utils.GetVNC(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 403)
			s.Error(err)
			s.T().Logf("successfully handled invalid project id to get vnc\n")
		}
	}
}

// TestVMAuthInvalidJWTGetVNC tests get vnc with invalid jwt
func (s *TestSuite) TestVMAuthInvalidJWTGetVNC() {
	armClient, err := clients.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.ProjectID)
	s.NoError(err)

	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			retCode, err = utils.GetVNC(armClient, appID, appWorkload.Id)
			s.Equal(retCode, 401)
			s.Error(err)
			s.T().Logf("successfully handled invalid JWT to get VNC\n")
		}
	}
}
