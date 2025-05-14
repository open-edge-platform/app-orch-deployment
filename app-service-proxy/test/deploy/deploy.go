// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deploy

import (
	"fmt"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

const (
	TestClusterID = "test7"
	retryCount    = 10
)

var dpConfigs = map[string]any{
	"nginx": map[string]any{
		"appNames":             []string{"nginx"},
		"deployPackage":        "nginx-app",
		"deployPackageVersion": "0.1.0",
		"profileName":          "testing-default",
	},
	"vm": map[string]any{
		"appNames":             []string{"librespeed-vm"},
		"deployPackage":        "librespeed-app",
		"deployPackageVersion": "1.0.0",
		"profileName":          "virtual-cluster",
	},
	"virt-extension": map[string]any{
		"appNames":             []string{"kubevirt", "cdi", "kube-helper"},
		"deployPackage":        "virtualization",
		"deployPackageVersion": "0.3.6",
		"profileName":          "with-software-emulation-profile-nosm",
	},
	"wordpress": map[string]any{
		"appNames":             []string{"wordpress"},
		"deployPackage":        "wordpress",
		"deployPackageVersion": "0.1.1",
		"profileName":          "testing",
	},
}

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

	/* Enable if you want to resue existing deployed app
    if deployID := findDeploymentIDByDisplayName(admClient, displayName); deployID != "" {
		fmt.Printf("Deployment exists. use it")
        deployApps, err := getDeployApps(admClient, deployID)
        if err != nil {
            return []*restClient.App{}, err
        }
		return deployApps, nil
	}*/

	err := deleteAndRetryUntilDeleted(admClient, displayName, retryCount, time.Duration(retryDelay)*time.Second)
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
