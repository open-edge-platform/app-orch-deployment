// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/deploy"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/container"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

func StartVirtualMachine(armClient *restClient.ClientWithResponses, appID, virtMachineID string) (int, error) {
	resp, err := armClient.VirtualMachineServiceStartVirtualMachineWithResponse(context.TODO(), appID, deploy.TestClusterID, virtMachineID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to start virtual machine: %v", string(resp.Body))
	}

	return resp.StatusCode(), nil
}

func StopVirtualMachine(armClient *restClient.ClientWithResponses, appID, virtMachineID string) (int, error) {
	resp, err := armClient.VirtualMachineServiceStopVirtualMachineWithResponse(context.TODO(), appID, deploy.TestClusterID, virtMachineID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to stop virtual machine: %v", string(resp.Body))
	}

	return resp.StatusCode(), nil
}

func RestartVirtualMachine(armClient *restClient.ClientWithResponses, appID, virtMachineID string) (int, error) {
	resp, err := armClient.VirtualMachineServiceRestartVirtualMachineWithResponse(context.TODO(), appID, deploy.TestClusterID, virtMachineID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to restart virtual machine: %v", string(resp.Body))
	}

	return resp.StatusCode(), nil
}

func GetVNC(armClient *restClient.ClientWithResponses, appID, virtMachineID string) (int, error) {
	resp, err := armClient.VirtualMachineServiceGetVNCWithResponse(context.TODO(), appID, deploy.TestClusterID, virtMachineID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to get VNC: %v", string(resp.Body))
	}

	return resp.StatusCode(), nil
}

func MethodGetVNC(verb, restServerURL, appID, token, projectID, virtMachineID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/virtual-machines/%s/%s/%s/vnc", restServerURL, appID, deploy.TestClusterID, virtMachineID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodVMStart(verb, restServerURL, appID, token, projectID, virtMachineID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/virtual-machines/%s/%s/%s/start", restServerURL, appID, deploy.TestClusterID, virtMachineID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodVMStop(verb, restServerURL, appID, token, projectID, virtMachineID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/virtual-machines/%s/%s/%s/stop", restServerURL, appID, deploy.TestClusterID, virtMachineID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodVMRestart(verb, restServerURL, appID, token, projectID, virtMachineID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/virtual-machines/%s/%s/%s/restart", restServerURL, appID, deploy.TestClusterID, virtMachineID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func GetVMStatus(armClient *restClient.ClientWithResponses, appID, virtMachineID, desiredState string) error {
	var (
		appName    string
		currState  string
		retryDelay = 10 * time.Second
		retryCount = 10
	)

	for range retryCount {
		appWorkloads, returnCode, err := container.AppWorkloadsList(armClient, appID)
		if err != nil || returnCode != 200 {
			return fmt.Errorf("failed to list app workloads: %v", err)
		}

		for _, appWorkload := range *appWorkloads {
			appName = appWorkload.Name
			currState = string(*appWorkload.VirtualMachine.Status.State)
			if appWorkload.Id == virtMachineID {
				if currState == desiredState {
					fmt.Printf("Waiting for VM %s state %s ---> %s\n", appName, currState, desiredState)
					return nil
				}
			}
		}

		fmt.Printf("Waiting for VM %s state %s ---> %s\n", appName, currState, desiredState)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("VM %s failed to reach desired state %s. Last known state: %s", appName, desiredState, currState)
}
