// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/container"
)

var getVncMethods = map[string]int{
	http.MethodPut:    405,
	http.MethodGet:    200,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPost:   405,
}

var startVMMethods = map[string]int{
	http.MethodPut:    200,
	http.MethodGet:    405,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPost:   405,
}

var stopVMMethods = map[string]int{
	http.MethodPut:    200,
	http.MethodGet:    405,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPost:   405,
}

var restartVMMethods = map[string]int{
	http.MethodPut:    200,
	http.MethodGet:    405,
	http.MethodDelete: 405,
	http.MethodPatch:  405,
	http.MethodPost:   405,
}

func (s *TestSuite) testVMMethod(appID string, appWorkloadID string, methods map[string]int, methodFunc func(string, string, string, string, string, string) (*http.Response, error), desiredState string) {
	for method, expectedStatus := range methods {
		res, err := methodFunc(method, s.ResourceRESTServerUrl, appID, s.token, s.projectID, appWorkloadID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)

		if expectedStatus == 200 && desiredState != "" {
			err = GetVMStatus(s.ArmClient, appID, appWorkloadID, desiredState)
			s.NoError(err)
		}

		s.T().Logf("method: %s (%d)\n", method, res.StatusCode)
	}
}

func (s *TestSuite) prepareVMForState(appID string, appWorkloadID string, currentState string, targetState string) {
	if currentState != targetState {
		var retCode int
		var err error

		if targetState == VMRunning {
			retCode, err = StartVirtualMachine(s.ArmClient, appID, appWorkloadID)
		} else if targetState == VMStopped {
			retCode, err = StopVirtualMachine(s.ArmClient, appID, appWorkloadID)
		}

		s.Equal(retCode, 200)
		s.NoError(err)
		err = GetVMStatus(s.ArmClient, appID, appWorkloadID, targetState)
		s.NoError(err)
	}
}

func (s *TestSuite) TestGetVNCResponseCodeValidation() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		for _, appWorkload := range *appWorkloads {
			s.testVMMethod(appID, appWorkload.Id, getVncMethods, MethodGetVNC, "")
		}
	}
}

func (s *TestSuite) TestStartVMResponseCodeValidation() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		for _, appWorkload := range *appWorkloads {
			currState := string(*appWorkload.VirtualMachine.Status.State)
			s.prepareVMForState(appID, appWorkload.Id, currState, VMStopped)
			s.testVMMethod(appID, appWorkload.Id, startVMMethods, MethodVMStart, VMRunning)
		}
	}
}

func (s *TestSuite) TestStopVMResponseCodeValidation() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		for _, appWorkload := range *appWorkloads {
			currState := string(*appWorkload.VirtualMachine.Status.State)
			s.prepareVMForState(appID, appWorkload.Id, currState, VMRunning)
			s.testVMMethod(appID, appWorkload.Id, stopVMMethods, MethodVMStop, VMStopped)
		}
	}
}

func (s *TestSuite) TestRestartVMResponseCodeValidation() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appWorkloads, retCode, err := container.AppWorkloadsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appWorkloads)

		for _, appWorkload := range *appWorkloads {
			currState := string(*appWorkload.VirtualMachine.Status.State)
			s.prepareVMForState(appID, appWorkload.Id, currState, VMRunning)
			s.testVMMethod(appID, appWorkload.Id, restartVMMethods, MethodVMRestart, VMRunning)
		}
	}
}
