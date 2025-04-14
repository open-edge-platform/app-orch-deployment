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
	appName              = "nginx"
	deployPackage        = "nginx-app"
	deployPackageVersion = "0.1.0"
	displayName          = "nginx"
	TestClusterID        = "demo-ruben5"
	profileName          = "testing-default"
	retryCount           = 10
	retryDelay           = 10 * time.Second
)

func CreateDeployment(admClient *restClient.ClientWithResponses) ([]*restClient.App, error) {
	err := deleteAndRetryUntilDeleted(admClient, displayName, retryCount, retryDelay)
	if err != nil {
		return []*restClient.App{}, err
	}
	fmt.Printf("Existing %s deployment deleted successfully\n", displayName)

	err = createTargetedDeployment(admClient, CreateDeploymentParams{
		ClusterID:      TestClusterID,
		DpName:         deployPackage,
		AppName:        appName,
		AppVersion:     deployPackageVersion,
		DisplayName:    displayName,
		ProfileName:    profileName,
		DeploymentType: "targeted",
	})
	if err != nil {
		return []*restClient.App{}, err
	}
	fmt.Printf("New %s deployment creation initiated\n", displayName)

	deployId, err := waitForDeploymentStatus(admClient, displayName, restClient.RUNNING, retryCount, retryDelay)
	if err != nil {
		return []*restClient.App{}, err
	}
	fmt.Printf("%s deployment is now in RUNNING status\n", displayName)

	deployApps, err := getDeployApps(admClient, deployId)
	if err != nil {
		return []*restClient.App{}, err
	}

	return deployApps, nil
}
