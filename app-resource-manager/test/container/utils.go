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
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

const (
	retryDelay = 10 * time.Second
	retryCount = 10
)

func AppWorkloadsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.AppWorkload, error) {
	resp, err := armClient.AppWorkloadServiceListAppWorkloadsWithResponse(context.TODO(), appID, deploy.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		return &[]restClient.AppWorkload{}, fmt.Errorf("failed to list app workloads: %v, status: %d", err, resp.StatusCode())
	}

	return resp.JSON200.AppWorkloads, nil
}

func AppEndpointsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.AppEndpoint, error) {
	resp, err := armClient.EndpointsServiceListAppEndpointsWithResponse(context.TODO(), appID, deploy.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		return &[]restClient.AppEndpoint{}, fmt.Errorf("failed to list app endpoints: %v, status: %d", err, resp.StatusCode())
	}

	return resp.JSON200.AppEndpoints, nil
}

func PodDelete(armClient *restClient.ClientWithResponses, namespace, podName, appID string) error {
	resp, err := armClient.PodServiceDeletePodWithResponse(context.TODO(), deploy.TestClusterID, namespace, podName)
	if err != nil || resp.StatusCode() != 200 {
		return fmt.Errorf("failed to delete pod: %v, status: %d", err, resp.StatusCode())
	}

	err = WaitPodDelete(armClient, appID)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	return nil
}

func MethodAppWorkloadsList(verb, restServerURL, appID, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/%s/%s", restServerURL, appID, deploy.TestClusterID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodAppEndpointsList(verb, restServerURL, appID, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/endpoints/%s/%s", restServerURL, appID, deploy.TestClusterID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodPodDelete(verb, restServerURL, namespace, podName, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/pods/%s/%s/%s/delete", restServerURL, deploy.TestClusterID, namespace, podName)
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
		appWorkloads, err := AppWorkloadsList(armClient, appID)
		if err != nil {
			return fmt.Errorf("failed to list app workloads: %v", err)
		}

		for _, appWorkload := range *appWorkloads {
			appName = appWorkload.Name
			currState = string(*appWorkload.Pod.Status.State)

			if appWorkload.Id.String() == workloadID {
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
		appWorkloads, err := AppWorkloadsList(armClient, appID)
		if err != nil {
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
