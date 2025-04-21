// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

func DeploymentsList(admClient *restClient.ClientWithResponses) (*[]restClient.Deployment, int, error) {
	resp, err := admClient.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.Deployment{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.Deployment{}, resp.StatusCode(), fmt.Errorf("failed to list deployments: %v", string(resp.Body))
	}

	return &resp.JSON200.Deployments, resp.StatusCode(), nil
}

func GetDeployment(admClient *restClient.ClientWithResponses, deployID string) (restClient.Deployment, int, error) {
	resp, err := admClient.DeploymentServiceGetDeploymentWithResponse(context.TODO(), deployID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return restClient.Deployment{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return restClient.Deployment{}, resp.StatusCode(), fmt.Errorf("failed to get deployment: %v", string(resp.Body))
	}

	return resp.JSON200.Deployment, resp.StatusCode(), nil
}
