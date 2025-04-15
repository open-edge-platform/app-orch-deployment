// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"context"
	"fmt"
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
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
