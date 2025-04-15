// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/list"
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

// TestVMMethods tests VM methods
func (s *TestSuite) TestGetVMMethod() {
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
			for method, expectedStatus := range getVncMethods {
				res, err := MethodGetVNC(method, s.ResourceRESTServerUrl, appID, s.token, s.projectID, appWorkload.Id.String())
				s.NoError(err)
				s.Equal(expectedStatus, res.StatusCode)
				s.T().Logf("get VNC method: %s (%d)\n", method, res.StatusCode)
			}
		}
	}
}

// TestStartVMMethod tests start VM methods
func (s *TestSuite) TestStartVMMethod() {
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
			// will get 400 if VM is already running
			currState := string(*appWorkload.VirtualMachine.Status.State)
			if currState != VMStopped {
				err = StopVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
				s.NoError(err)
				s.T().Logf("stop VM pod %s\n", appWorkload.Name)

				err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMStopped)
				s.NoError(err)
			}

			for method, expectedStatus := range startVMMethods {
				res, err := MethodStartVNC(method, s.ResourceRESTServerUrl, appID, s.token, s.projectID, appWorkload.Id.String())
				s.NoError(err)
				s.Equal(expectedStatus, res.StatusCode)

				if expectedStatus == 200 {
					err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMRunning)
					s.NoError(err)
				}

				s.T().Logf("start VM method: %s (%d)\n", method, res.StatusCode)
			}
		}
	}
}

// TestStopVMMethod tests stop VM methods
func (s *TestSuite) TestStopVMMethod() {
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
			// will get 400 if VM is not already running
			currState := string(*appWorkload.VirtualMachine.Status.State)
			if currState != VMRunning {
				err = StartVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMRunning)
				s.NoError(err)
			}

			for method, expectedStatus := range stopVMMethods {
				res, err := MethodStopVNC(method, s.ResourceRESTServerUrl, appID, s.token, s.projectID, appWorkload.Id.String())
				s.NoError(err)
				s.Equal(expectedStatus, res.StatusCode)

				if expectedStatus == 200 {
					err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMStopped)
					s.NoError(err)
				}

				s.T().Logf("stop VM method: %s (%d)\n", method, res.StatusCode)
			}
		}
	}
}

// TestRestartVMMethod tests restart VM methods
func (s *TestSuite) TestRestartVMMethod() {
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
			// will get 400 if VM is not already running
			currState := string(*appWorkload.VirtualMachine.Status.State)
			if currState != VMRunning {
				err = StartVirtualMachine(s.ArmClient, appID, appWorkload.Id.String())
				s.NoError(err)
				s.T().Logf("start VM pod %s\n", appWorkload.Name)

				err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMRunning)
				s.NoError(err)
			}

			for method, expectedStatus := range restartVMMethods {
				res, err := MethodRestartVNC(method, s.ResourceRESTServerUrl, appID, s.token, s.projectID, appWorkload.Id.String())
				s.NoError(err)
				s.Equal(expectedStatus, res.StatusCode)

				if expectedStatus == 200 {
					err = GetVNCStatus(s.ArmClient, appID, appWorkload.Id.String(), VMRunning)
					s.NoError(err)
				}

				s.T().Logf("restart VM method: %s (%d)\n", method, res.StatusCode)
			}
		}
	}
}
