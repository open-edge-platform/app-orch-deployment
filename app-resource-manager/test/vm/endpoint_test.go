// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/container"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/shared"
	"net/http"

	"fmt"
)

// ensureVMState ensures the VM is in the desired state before proceeding with tests
// This is a helper function that simplifies VM state management across tests
// Parameters:
//   - appID: The application ID
//   - workloadID: The workload ID for the VM
//   - desiredState: The state we want the VM to be in (VMRunning or VMStopped)
//
// Returns:
//   - error: Any error that occurred during the state transition
func (s *TestSuite) ensureVMState(appID string, workloadID string, desiredState string) error {
	// Get current VM state using AppWorkloadsList
	appWorkloads, _, err := container.AppWorkloadsList(s.ArmClient, appID)
	if err != nil {
		return err
	}

	// Find the specific workload by ID in the list of workloads
	var targetWorkload *armClient.AppWorkload
	for _, wl := range *appWorkloads {
		if wl.Id == workloadID {
			targetWorkload = &wl
			break
		}
	}

	// Return error if workload doesn't exist
	if targetWorkload == nil {
		return fmt.Errorf("workload not found: %s", workloadID)
	}

	// Check if VM is already in the desired state to avoid unnecessary operations
	currState := string(*targetWorkload.VirtualMachine.Status.State)
	if currState == desiredState {
		// VM already in desired state, do nothing
		return nil
	}

	// Transition VM to the desired state based on what's needed
	switch desiredState {
	case VMRunning:
		// Start the VM if we need it to be running
		retCode, err := shared.StartVirtualMachine(s.ArmClient, appID, workloadID)
		if err != nil || retCode != http.StatusOK {
			return err
		}
	case VMStopped:
		// Stop the VM if we need it to be stopped
		retCode, err := shared.StopVirtualMachine(s.ArmClient, appID, workloadID)
		if err != nil || retCode != http.StatusOK {
			return err
		}
	}

	// Wait for the VM to reach the desired state and verify
	return shared.GetVMStatus(s.ArmClient, appID, workloadID, desiredState)
}

// performVMAction performs the specified action on a VM and verifies the result
// This is a high-level helper that handles state preconditions and verification
// Parameters:
//   - appID: The application ID
//   - workloadID: The workload ID for the VM
//   - workloadName: The name of the workload (for logging)
//   - action: The action to perform ("start", "stop", or "restart")
//   - requiredState: The state the VM must be in before performing the action
//   - expectedState: The state the VM should be in after the action completes
func (s *TestSuite) performVMAction(appID string, workloadID string, workloadName string, action string, requiredState string, expectedState string) {
	var retCode int
	var err error

	// First, ensure VM is in the required state before attempting the action
	// For example, VM must be stopped before starting, running before stopping/restarting
	err = s.ensureVMState(appID, workloadID, requiredState)
	s.NoError(err)

	// Perform the requested action based on the action parameter
	switch action {
	case "start":
		retCode, err = shared.StartVirtualMachine(s.ArmClient, appID, workloadID)
		s.T().Logf("start VM pod %s\n", workloadName)
	case "stop":
		retCode, err = shared.StopVirtualMachine(s.ArmClient, appID, workloadID)
		s.T().Logf("stop VM pod %s\n", workloadName)
	case "restart":
		retCode, err = shared.RestartVirtualMachine(s.ArmClient, appID, workloadID)
		s.T().Logf("restart VM pod %s\n", workloadName)
	}

	// Verify the API call was successful
	s.Equal(retCode, http.StatusOK)
	s.NoError(err)

	// Wait and verify that the VM reached the expected state after the action
	err = shared.GetVMStatus(s.ArmClient, appID, workloadID, expectedState)
	s.NoError(err)
}

// TestGetVNC tests get vnc endpoint
// This test verifies that the VNC endpoint returns a successful response
func (s *TestSuite) TestGetVNC() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// app workload len should be 1
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			retCode, err = shared.GetVNC(s.ArmClient, appID, appWorkload.Id)
			s.Equal(retCode, 200)
			s.NoError(err)
			s.T().Logf("get VM pod %s\n", appWorkload.Name)
		}
	}
}

// TestVMStart tests VM start endpoint
// This test verifies that:
// 1. The VM can be started successfully from a stopped state
// 2. The VM reaches the running state after being started
func (s *TestSuite) TestVMStart() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		// Get all workloads for the application
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// Each app should have exactly one workload
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			// Perform the start action, ensuring VM is first stopped, and should end up running
			s.performVMAction(appID, appWorkload.Id, appWorkload.Name, "start", VMStopped, VMRunning)
		}
	}
}

// TestVMStop tests VM stop endpoint
// This test verifies that:
// 1. The VM can be stopped successfully from a running state
// 2. The VM reaches the stopped state after being stopped
func (s *TestSuite) TestVMStop() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		// Get all workloads for the application
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// Each app should have exactly one workload
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			// Perform the stop action, ensuring VM is first running, and should end up stopped
			s.performVMAction(appID, appWorkload.Id, appWorkload.Name, "stop", VMRunning, VMStopped)
		}
	}
}

// TestVMRestart tests VM restart endpoint
// This test verifies that:
// 1. The VM can be restarted successfully from a running state
// 2. The VM remains in a running state after being restarted
func (s *TestSuite) TestVMRestart() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		// Get all workloads for the application
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		// Each app should have exactly one workload
		s.Assert().Equal(1, len(*appWorkloads), "invalid app workloads len: %+v expected len 1", len(*appWorkloads))

		for _, appWorkload := range *appWorkloads {
			// For restart, we need to ensure it's already running and check if it's still running after
			currState := string(*appWorkload.VirtualMachine.Status.State)
			// Perform the restart action, ensuring VM is first running, and should maintain the same state
			s.performVMAction(appID, appWorkload.Id, appWorkload.Name, "restart", VMRunning, currState)
		}
	}
}
