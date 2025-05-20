// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

const (
	retryDelay = 10 * time.Second
	retryCount = 10
)

func AppWorkloadsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.AppWorkload, int, error) {
	resp, err := armClient.AppWorkloadServiceListAppWorkloadsWithResponse(context.TODO(), appID, utils.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.AppWorkload{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.AppWorkload{}, resp.StatusCode(), fmt.Errorf("failed to list app workloads: %v", string(resp.Body))
	}

	return resp.JSON200.AppWorkloads, resp.StatusCode(), nil
}

func AppEndpointsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.AppEndpoint, int, error) {
	resp, err := armClient.EndpointsServiceListAppEndpointsWithResponse(context.TODO(), appID, utils.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.AppEndpoint{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.AppEndpoint{}, resp.StatusCode(), fmt.Errorf("failed to list app endpoints: %v", string(resp.Body))
	}

	return resp.JSON200.AppEndpoints, resp.StatusCode(), nil
}

func PodDelete(armClient *restClient.ClientWithResponses, namespace, podName, appID string) (int, error) {
	resp, err := armClient.PodServiceDeletePodWithResponse(context.TODO(), utils.TestClusterID, namespace, podName)
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
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/%s/%s", restServerURL, appID, utils.TestClusterID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodAppEndpointsList(verb, restServerURL, appID, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/endpoints/%s/%s", restServerURL, appID, utils.TestClusterID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodPodDelete(verb, restServerURL, namespace, podName, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/pods/%s/%s/%s/delete", restServerURL, utils.TestClusterID, namespace, podName)
	res, err := utils.CallMethod(url, verb, token, projectID)
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

	for range retryCount {
		appWorkloads, returnCode, err := AppWorkloadsList(armClient, appID)
		if err != nil || returnCode != 200 {
			return fmt.Errorf("failed to list app workloads: %v", err)
		}

		for _, appWorkload := range *appWorkloads {
			appName = appWorkload.Name
			currState = string(*appWorkload.Pod.Status.State)

			if appWorkload.Id == workloadID {
				if currState == desiredState {
					fmt.Printf("Waiting for POD %s state %s ---> %s\n", appName, currState, desiredState)
					return nil
				}
			}
		}

		fmt.Printf("Waiting for POD %s state %s ---> %s\n", appName, currState, desiredState)
		time.Sleep(retryDelay)
	}

	return nil
}

func WaitPodDelete(armClient *restClient.ClientWithResponses, appID string) error {
	for range retryCount {
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
		time.Sleep(retryDelay)
	}

	return nil
}
