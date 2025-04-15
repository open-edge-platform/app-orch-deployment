// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"context"
	"fmt"
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/deploy"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
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
