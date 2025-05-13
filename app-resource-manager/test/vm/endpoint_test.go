// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/container"
	"net/http"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"

	"fmt"
)

// ensureVMState ensures the VM is in the desired state before proceeding
// If the VM is already in the desired state, it does nothing
// Otherwise, it transitions the VM to the desired state
func (s *TestSuite) ensureVMState(appID string, workloadID string, desiredState string) error {
	// Get current VM state using AppWorkloadsList
	appWorkloads, _, err := container.AppWorkloadsList(s.armClient, appID)
	if err != nil {
		return err
	}

	// Find the specific workload by ID
	var targetWorkload *armClient.AppWorkload
	for _, wl := range *appWorkloads {
		if wl.Id == workloadID {
			targetWorkload = &wl
			break
		}
	}

	if targetWorkload == nil {
		return fmt.Errorf("workload not found: %s", workloadID)
	}

	currState := string(*targetWorkload.VirtualMachine.Status.State)
	if currState == desiredState {
		// VM already in desired state, do nothing
		return nil
	}

	// Transition to desired state
	switch desiredState {
	case VMRunning:
		retCode, err := StartVirtualMachine(s.armClient, appID, workloadID)
		if err != nil || retCode != http.StatusOK {
			return err
		}
	case VMStopped:
		retCode, err := StopVirtualMachine(s.armClient, appID, workloadID)
		if err != nil || retCode != http.StatusOK {
			return err
		}
	}

	// Wait for the VM to reach the desired state
	return GetVMStatus(s.armClient, appID, workloadID, desiredState)
}

// performVMAction performs the specified action on a VM and verifies the result
func (s *TestSuite) performVMAction(appID string, workloadID string, workloadName string, action string, requiredState string, expectedState string) {
	var retCode int
	var err error

	// Ensure VM is in required state before action
	err = s.ensureVMState(appID, workloadID, requiredState)
	s.NoError(err)

	// Perform the requested action
	switch action {
	case "start":
		retCode, err = StartVirtualMachine(s.armClient, appID, workloadID)
		s.T().Logf("start VM pod %s\n", workloadName)
	case "stop":
		retCode, err = StopVirtualMachine(s.armClient, appID, workloadID)
		s.T().Logf("stop VM pod %s\n", workloadName)
	case "restart":
		retCode, err = RestartVirtualMachine(s.armClient, appID, workloadID)
		s.T().Logf("restart VM pod %s\n", workloadName)
	}

	s.Equal(retCode, http.StatusOK)
	s.NoError(err)

	// Verify VM reached expected state
	err = GetVMStatus(s.armClient, appID, workloadID, expectedState)
	s.NoError(err)
}

// TestGetVNC tests get vnc endpoint
func (s *TestSuite) TestGetVNC() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.armClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			retCode, err = GetVNC(s.armClient, appID, appWorkload.Id)
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
		appWorkloads, retCode, err := container.AppWorkloadsList(s.armClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			s.performVMAction(appID, appWorkload.Id, appWorkload.Name, "start", VMStopped, VMRunning)
		}
	}
}

// TestVMStop tests VM stop endpoint
func (s *TestSuite) TestVMStop() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.armClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			s.performVMAction(appID, appWorkload.Id, appWorkload.Name, "stop", VMRunning, VMStopped)
		}
	}
}

// TestVMRestart tests VM restart endpoint
func (s *TestSuite) TestVMRestart() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.armClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			// For restart, we need to know what the current state is to check at the end
			currState := string(*appWorkload.VirtualMachine.Status.State)
			s.performVMAction(appID, appWorkload.Id, appWorkload.Name, "restart", VMRunning, currState)
		}
	}
}
