// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

// HTTP method maps for various VM operations
// These maps define the expected HTTP status codes for different HTTP methods
// for each VM operation endpoint

// Map of HTTP methods and their expected status codes for VNC endpoint
var getVncMethods = map[string]int{
	http.MethodPut:    405, // Method not allowed
	http.MethodGet:    200, // OK - this is the only allowed method
	http.MethodDelete: 405, // Method not allowed
	http.MethodPatch:  405, // Method not allowed
	http.MethodPost:   405, // Method not allowed
}

// Map of HTTP methods and their expected status codes for VM start endpoint
var startVMMethods = map[string]int{
	http.MethodPut:    200, // OK - this is the only allowed method
	http.MethodGet:    405, // Method not allowed
	http.MethodDelete: 405, // Method not allowed
	http.MethodPatch:  405, // Method not allowed
	http.MethodPost:   405, // Method not allowed
}

// Map of HTTP methods and their expected status codes for VM stop endpoint
var stopVMMethods = map[string]int{
	http.MethodPut:    200, // OK - this is the only allowed method
	http.MethodGet:    405, // Method not allowed
	http.MethodDelete: 405, // Method not allowed
	http.MethodPatch:  405, // Method not allowed
	http.MethodPost:   405, // Method not allowed
}

// Map of HTTP methods and their expected status codes for VM restart endpoint
var restartVMMethods = map[string]int{
	http.MethodPut:    200, // OK - this is the only allowed method
	http.MethodGet:    405, // Method not allowed
	http.MethodDelete: 405, // Method not allowed
	http.MethodPatch:  405, // Method not allowed
	http.MethodPost:   405, // Method not allowed
}

// testVMMethod tests various HTTP methods against VM endpoints
// This function tests all HTTP methods (GET, PUT, POST, DELETE, PATCH) against a specific VM endpoint
// Parameters:
//   - appID: The application ID
//   - appWorkloadID: The workload ID for the VM
//   - methods: Map of HTTP methods and their expected status codes
//   - methodFunc: Function to call the specific VM operation with the given HTTP method
//   - desiredState: The state the VM should be in after successful operations (200 response)
func (s *TestSuite) testVMMethod(appID string, appWorkloadID string, methods map[string]int, methodFunc func(string, string, string, string, string, string) (*http.Response, error), desiredState string) {
	for method, expectedStatus := range methods {
		// Call the endpoint with the current HTTP method
		res, err := methodFunc(method, s.ResourceRESTServerUrl, appID, s.Token, s.ProjectID, appWorkloadID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)

		// If the operation should be successful (200) and we expect a state change,
		// verify the VM reached the expected state
		if expectedStatus == http.StatusOK && desiredState != "" {
			err = utils.GetVMStatus(s.ArmClient, appID, appWorkloadID, desiredState)
			s.NoError(err)
		}

		s.T().Logf("method: %s (%d)\n", method, res.StatusCode)
	}
}

// prepareVMForState prepares the VM to be in a specific state
// This is a wrapper around ensureVMState for backward compatibility
// Parameters:
//   - appID: The application ID
//   - appWorkloadID: The workload ID for the VM
//   - currentState: Current state of the VM (unused but kept for API compatibility)
//   - targetState: The state we want the VM to be in
func (s *TestSuite) prepareVMForState(appID string, appWorkloadID string, currentState string, targetState string) {
	// We can simply use ensureVMState which already handles the state transition logic
	err := s.ensureVMState(appID, appWorkloadID, targetState)
	s.NoError(err)
}

// TestGetVNCResponseCodeValidation tests the VNC endpoint with all HTTP methods
// This test verifies that:
// 1. GET method returns 200 (success)
// 2. All other HTTP methods return 405 (method not allowed)
func (s *TestSuite) TestGetVNCResponseCodeValidation() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		// Get all workloads for the application
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		for _, appWorkload := range *appWorkloads {
			// Test all HTTP methods on the VNC endpoint
			// No desired state is specified ("") since VNC doesn't change VM state
			s.testVMMethod(appID, appWorkload.Id, getVncMethods, utils.MethodGetVNC, "")
		}
	}
}

// TestStartVMResponseCodeValidation tests the Start VM endpoint with all HTTP methods
// This test verifies that:
// 1. PUT method returns 200 (success) and changes VM state to running
// 2. All other HTTP methods return 405 (method not allowed)
func (s *TestSuite) TestStartVMResponseCodeValidation() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		// Get all workloads for the application
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		for _, appWorkload := range *appWorkloads {
			// Make sure VM is stopped before testing start methods
			// This ensures that the PUT method has a state to change
			err := s.ensureVMState(appID, appWorkload.Id, VMStopped)
			s.NoError(err)

			// Test all HTTP methods on the start endpoint
			// The desired state after successful operation is VMRunning
			s.testVMMethod(appID, appWorkload.Id, startVMMethods, utils.MethodVMStart, VMRunning)
		}
	}
}

// TestStopVMResponseCodeValidation tests the Stop VM endpoint with all HTTP methods
// This test verifies that:
// 1. PUT method returns 200 (success) and changes VM state to stopped
// 2. All other HTTP methods return 405 (method not allowed)
func (s *TestSuite) TestStopVMResponseCodeValidation() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		// Get all workloads for the application
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		for _, appWorkload := range *appWorkloads {
			// Make sure VM is running before testing stop methods
			// This ensures that the PUT method has a state to change
			err := s.ensureVMState(appID, appWorkload.Id, VMRunning)
			s.NoError(err)

			// Test all HTTP methods on the stop endpoint
			// The desired state after successful operation is VMStopped
			s.testVMMethod(appID, appWorkload.Id, stopVMMethods, utils.MethodVMStop, VMStopped)
		}
	}
}

// TestRestartVMResponseCodeValidation tests the Restart VM endpoint with all HTTP methods
// This test verifies that:
// 1. PUT method returns 200 (success) and maintains VM in running state
// 2. All other HTTP methods return 405 (method not allowed)
func (s *TestSuite) TestRestartVMResponseCodeValidation() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		// Get all workloads for the application
		appWorkloads, retCode, err := utils.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, http.StatusOK)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		for _, appWorkload := range *appWorkloads {
			// Make sure VM is running before testing restart methods
			// VM must be running to be restarted
			err := s.ensureVMState(appID, appWorkload.Id, VMRunning)
			s.NoError(err)

			// Test all HTTP methods on the restart endpoint
			// The desired state after successful operation is VMRunning
			s.testVMMethod(appID, appWorkload.Id, restartVMMethods, utils.MethodVMRestart, VMRunning)
		}
	}
}
