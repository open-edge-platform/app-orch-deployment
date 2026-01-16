// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Test utility package
package utils

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	testTypes "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
)

func CallMethod(url, verb, token, projectID string) (*http.Response, error) {
	req, err := http.NewRequest(verb, url, nil)
	if err != nil {
		return nil, err
	}

	auth.AddRestAuthHeader(req, token, projectID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, err
}

func StartVirtualMachine(armClient *restClient.ClientWithResponses, appID, virtMachineID string) (int, error) {
	uuidVal, err := uuid.Parse(virtMachineID)
	if err != nil {
		return 0, fmt.Errorf("invalid UUID format: %v", err)
	}
	resp, err := armClient.ResourceV2VirtualMachineServiceStartVirtualMachineWithResponse(context.TODO(), appID, testTypes.TestClusterID, uuidVal)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to start virtual machine: %v", string(resp.Body))
	}

	return resp.StatusCode(), nil
}

func StopVirtualMachine(armClient *restClient.ClientWithResponses, appID, virtMachineID string) (int, error) {
	uuidVal, err := uuid.Parse(virtMachineID)
	if err != nil {
		return 0, fmt.Errorf("invalid UUID format: %v", err)
	}
	resp, err := armClient.ResourceV2VirtualMachineServiceStopVirtualMachineWithResponse(context.TODO(), appID, testTypes.TestClusterID, uuidVal)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to stop virtual machine: %v", string(resp.Body))
	}

	return resp.StatusCode(), nil
}

func RestartVirtualMachine(armClient *restClient.ClientWithResponses, appID, virtMachineID string) (int, error) {
	uuidVal, err := uuid.Parse(virtMachineID)
	if err != nil {
		return 0, fmt.Errorf("invalid UUID format: %v", err)
	}
	resp, err := armClient.ResourceV2VirtualMachineServiceRestartVirtualMachineWithResponse(context.TODO(), appID, testTypes.TestClusterID, uuidVal)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to restart virtual machine: %v", string(resp.Body))
	}

	return resp.StatusCode(), nil
}

func GetVNC(armClient *restClient.ClientWithResponses, appID, virtMachineID string) (int, error) {
	uuidVal, err := uuid.Parse(virtMachineID)
	if err != nil {
		return 0, fmt.Errorf("invalid UUID format: %v", err)
	}
	resp, err := armClient.ResourceV2VirtualMachineServiceGetVNCWithResponse(context.TODO(), appID, testTypes.TestClusterID, uuidVal)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to get VNC: %v", string(resp.Body))
	}

	return resp.StatusCode(), nil
}

func MethodGetVNC(verb, restServerURL, appID, token, projectID, virtMachineID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/virtual-machines/%s/%s/%s/vnc", restServerURL, appID, testTypes.TestClusterID, virtMachineID)
	res, err := CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodVMStart(verb, restServerURL, appID, token, projectID, virtMachineID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/virtual-machines/%s/%s/%s/start", restServerURL, appID, testTypes.TestClusterID, virtMachineID)
	res, err := CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodVMStop(verb, restServerURL, appID, token, projectID, virtMachineID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/virtual-machines/%s/%s/%s/stop", restServerURL, appID, testTypes.TestClusterID, virtMachineID)
	res, err := CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodVMRestart(verb, restServerURL, appID, token, projectID, virtMachineID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/virtual-machines/%s/%s/%s/restart", restServerURL, appID, testTypes.TestClusterID, virtMachineID)
	res, err := CallMethod(url, verb, token, projectID)
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
		appWorkloads, returnCode, err := AppWorkloadsList(armClient, appID)
		if err != nil || returnCode != 200 {
			return fmt.Errorf("failed to list app workloads: %v", err)
		}

		for _, appWorkload := range *appWorkloads {
			if appWorkload.Name != nil {
				appName = *appWorkload.Name
			}
			if appWorkload.Type != nil && *appWorkload.Type == restClient.TYPEVIRTUALMACHINE {
				if vmWorkload, err := appWorkload.AsResourceV2AppWorkload1(); err == nil {
					currState = string(*vmWorkload.VirtualMachine.Status.State)
					if appWorkload.Id != nil && appWorkload.Id.String() == virtMachineID {
						if currState == desiredState {
							fmt.Printf("Waiting for VM %s state %s ---> %s\n", appName, currState, desiredState)
							return nil
						}
					}
				}
			}
		}

		fmt.Printf("Waiting for VM %s state %s ---> %s\n", appName, currState, desiredState)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("VM %s failed to reach desired state %s. Last known state: %s", appName, desiredState, currState)
}

func AppWorkloadsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.ResourceV2AppWorkload, int, error) {
	resp, err := armClient.ResourceV2AppWorkloadServiceListAppWorkloadsWithResponse(context.TODO(), appID, testTypes.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.ResourceV2AppWorkload{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.ResourceV2AppWorkload{}, resp.StatusCode(), fmt.Errorf("failed to list app workloads: %v", string(resp.Body))
	}

	return resp.JSON200.AppWorkloads, resp.StatusCode(), nil
}

func AppEndpointsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.ResourceV2AppEndpoint, int, error) {
	resp, err := armClient.ResourceV2EndpointsServiceListAppEndpointsWithResponse(context.TODO(), appID, testTypes.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.ResourceV2AppEndpoint{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.ResourceV2AppEndpoint{}, resp.StatusCode(), fmt.Errorf("failed to list app endpoints: %v", string(resp.Body))
	}

	return resp.JSON200.AppEndpoints, resp.StatusCode(), nil
}

func PodDelete(armClient *restClient.ClientWithResponses, namespace, podName, appID string) (int, error) {
	resp, err := armClient.ResourceV2PodServiceDeletePodWithResponse(context.TODO(), testTypes.TestClusterID, namespace, podName)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return resp.StatusCode(), fmt.Errorf("failed to delete pod: %v", string(resp.Body))
	}

	err = WaitPodDelete(armClient, appID)
	if err != nil {
		return resp.StatusCode(), fmt.Errorf("error: %v", err)
	}

	return resp.StatusCode(), nil
}

func MethodAppWorkloadsList(verb, restServerURL, appID, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/%s/%s", restServerURL, appID, testTypes.TestClusterID)
	res, err := CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodAppEndpointsList(verb, restServerURL, appID, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/endpoints/%s/%s", restServerURL, appID, testTypes.TestClusterID)
	res, err := CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodPodDelete(verb, restServerURL, namespace, podName, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/pods/%s/%s/%s/delete", restServerURL, testTypes.TestClusterID, namespace, podName)
	res, err := CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func GetPodStatus(armClient *restClient.ClientWithResponses, appID, workloadID, desiredState string) error {
	var (
		appName   string
		currState string
	)

	for range testTypes.RetryCount {
		appWorkloads, returnCode, err := AppWorkloadsList(armClient, appID)
		if err != nil || returnCode != 200 {
			return fmt.Errorf("failed to list app workloads: %v", err)
		}

		for _, appWorkload := range *appWorkloads {
			if appWorkload.Name != nil {
				appName = *appWorkload.Name
			}
			if appWorkload.Type != nil && *appWorkload.Type == restClient.TYPEPOD {
				if podWorkload, err := appWorkload.AsResourceV2AppWorkload0(); err == nil {
					currState = string(*podWorkload.Pod.Status.State)

					if appWorkload.Id != nil && appWorkload.Id.String() == workloadID {
						if currState == desiredState {
							fmt.Printf("Waiting for POD %s state %s ---> %s\n", appName, currState, desiredState)
							return nil
						}
					}
				}
			}
		}

		fmt.Printf("Waiting for POD %s state %s ---> %s\n", appName, currState, desiredState)
		time.Sleep(testTypes.RetryDelay)
	}

	return nil
}

func WaitPodDelete(armClient *restClient.ClientWithResponses, appID string) error {
	for range testTypes.RetryCount {
		appWorkloads, returnCode, err := AppWorkloadsList(armClient, appID)
		if err != nil || returnCode != 200 {
			return fmt.Errorf("failed to list app workloads: %v", err)
		}

		totalPods := len(*appWorkloads)

		if totalPods == 1 {
			fmt.Printf("Waiting for previous POD to delete (total %d)\n", totalPods)
			return nil
		}

		fmt.Printf("Waiting for previous POD to delete (total %d)\n", totalPods)
		time.Sleep(testTypes.RetryDelay)
	}

	return nil
}
