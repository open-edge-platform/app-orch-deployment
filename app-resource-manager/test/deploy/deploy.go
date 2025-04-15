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
	TestClusterID = "demo-cluster"
	retryCount    = 10
)

var appConfigs = map[string]any{
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

func CreateDeployment(admClient *restClient.ClientWithResponses, dpName string, displayName string, retryDelay int) ([]*restClient.App, error) {
	useDP := appConfigs[dpName].(map[string]any)

	// Check if virt-extension DP is already running, do not recreate a new one
	if dpName == "virt-extension" {
		deployments, err := getDeploymentPerCluster(admClient)
		if err != nil {
			return []*restClient.App{}, fmt.Errorf("failed to get deployments: %v", err)
		}
		for _, deployment := range deployments {
			if *deployment.DeploymentDisplayName == displayName && *deployment.Status.State == "RUNNING" {
				fmt.Printf("Deployment %s already exists in cluster %s, skipping creation\n", useDP["deployPackage"], TestClusterID)
				return []*restClient.App{}, nil
			}
		}
	}

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
