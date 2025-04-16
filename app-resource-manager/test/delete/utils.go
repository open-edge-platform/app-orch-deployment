// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/list"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

const (
	retryDelay = 10 * time.Second
	retryCount = 10
)

func PodDelete(armClient *restClient.ClientWithResponses, namespace, podName string) error {
	resp, err := armClient.PodServiceDeletePodWithResponse(context.TODO(), deploy.TestClusterID, namespace, podName)
	if err != nil || resp.StatusCode() != 200 {
		return fmt.Errorf("failed to delete pod: %v, status: %d", err, resp.StatusCode())
	}

	return nil
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
		appWorkloads, err := list.AppWorkloadsList(armClient, appID)
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
		appWorkloads, err := list.AppWorkloadsList(armClient, appID)
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
