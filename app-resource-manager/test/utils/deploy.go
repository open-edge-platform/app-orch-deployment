// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

func CreateDeployment(admClient *restClient.ClientWithResponses, dpPackageName string, displayName string, retryDelay int) ([]*restClient.App, error) {
	if dpConfigs[dpPackageName] == nil {
		return []*restClient.App{}, fmt.Errorf("deployment package %s not found in configuration", dpPackageName)
	}

	useDP := dpConfigs[dpPackageName].(map[string]any)

	// Check if virt-extension DP is already running, do not recreate a new one
	if dpPackageName == "virt-extension" {
		deployments, retCode, err := getDeploymentPerCluster(admClient)
		if err != nil || retCode != 200 {
			return []*restClient.App{}, fmt.Errorf("failed to get deployments per cluster: %v, status code: %d", err, retCode)
		}
		for _, deployment := range deployments {
			if *deployment.DeploymentDisplayName == displayName && *deployment.Status.State == "RUNNING" {
				fmt.Printf("Deployment %s already exists in cluster %s, skipping creation\n", useDP["deployPackage"], TestClusterID)
				return []*restClient.App{}, nil
			}
		}
	}

	err := DeleteAndRetryUntilDeleted(admClient, displayName, retryCount, time.Duration(retryDelay)*time.Second)
	if err != nil {
		return []*restClient.App{}, err
	}

	err = createTargetedDeployment(admClient, CreateDeploymentParams{
		ClusterID:      TestClusterID,
		DpName:         useDP["deployPackage"].(string),
		AppNames:       useDP["appNames"].([]string),
		AppVersion:     useDP["deployPackageVersion"].(string),
		DisplayName:    displayName,
		ProfileName:    useDP["profileName"].(string),
		DeploymentType: "targeted",
	})
	if err != nil {
		return []*restClient.App{}, err
	}
	fmt.Printf("New %s deployment creation initiated\n", displayName)

	deployID, err := waitForDeploymentStatus(admClient, displayName, restClient.RUNNING, retryCount, time.Duration(retryDelay)*time.Second)
	if err != nil {
		return []*restClient.App{}, err
	}
	fmt.Printf("%s deployment is now in RUNNING status\n", displayName)

	deployApps, err := getDeployApps(admClient, deployID)
	if err != nil {
		return []*restClient.App{}, err
	}

	return deployApps, nil
}
