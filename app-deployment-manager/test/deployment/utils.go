// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
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

func CopyOriginalDpConfig(originalDpConfigs map[string]any) map[string]any {
	tempDpConfigs := make(map[string]any)
	for key, value := range originalDpConfigs {
		if nestedMap, ok := value.(map[string]any); ok {
			deepCopy := make(map[string]any)
			for nestedKey, nestedValue := range nestedMap {
				if slice, ok := nestedValue.([]string); ok {
					copiedSlice := make([]string, len(slice))
					copy(copiedSlice, slice)
					deepCopy[nestedKey] = copiedSlice
				} else {
					deepCopy[nestedKey] = nestedValue
				}
			}
			tempDpConfigs[key] = deepCopy
		} else {
			tempDpConfigs[key] = value
		}
	}

	return tempDpConfigs
}

func ResetThenChangeDpConfig(dpConfigName string, key string, value any, originalDpConfigs map[string]any) error {
	utils.DpConfigs = CopyOriginalDpConfig(originalDpConfigs)

	if dpConfig, ok := utils.DpConfigs[dpConfigName].(map[string]any); ok {
		dpConfig[key] = value
		utils.DpConfigs[dpConfigName] = dpConfig
	} else {
		return fmt.Errorf("failed to assert type of deploy.DpConfigs[%s] as map[string]any", dpConfigName)
	}
	return nil
}
