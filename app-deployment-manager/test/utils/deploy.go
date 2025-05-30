// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

const (
	TestClusterID = "demo-cluster"
	retryCount    = 20
)

var DpConfigs = map[string]any{
	"nginx": map[string]any{
		"appNames":             []string{"nginx"},
		"deployPackage":        "nginx-app",
		"deployPackageVersion": "0.1.0",
		"profileName":          "testing-default",
		"clusterId":            TestClusterID,
		"labels":               map[string]string{"color": "blue"},

		"overrideValues": []map[string]any{},
	},
	"vm": map[string]any{
		"appNames":             []string{"librespeed-vm"},
		"deployPackage":        "librespeed-app",
		"deployPackageVersion": "1.0.0",
		"profileName":          "virtual-cluster",
		"clusterId":            TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
	"virt-extension": map[string]any{
		"appNames":             []string{"kubevirt", "cdi", "kube-helper"},
		"deployPackage":        "virtualization",
		"deployPackageVersion": "0.3.6",
		"profileName":          "with-software-emulation-profile-nosm",
		"clusterId":            TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
	"wordpress": map[string]any{
		"appNames":             []string{"wordpress"},
		"deployPackage":        "wordpress",
		"deployPackageVersion": "0.1.0",
		"profileName":          "testing",
		"clusterId":            TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
}

const (
	// DeploymentTypeTargeted represents the targeted deployment type
	DeploymentTypeTargeted = "targeted"
	// DeploymentTypeAutoScaling represents the auto-scaling deployment type
	DeploymentTypeAutoScaling = "auto-scaling"

	// AppWordpress represents the WordPress application name
	AppWordpress = "wordpress"
	// AppNginx represents the Nginx application name
	AppNginx = "nginx"

	// DeploymentTimeout represents the timeout in seconds for deployment operations
	DeploymentTimeout = 20
)

type CreateDeploymentParams struct {
	DpName         string
	AppNames       []string
	AppVersion     string
	DisplayName    string
	ProfileName    string
	ClusterID      string
	DeploymentType string
	OverrideValues []map[string]any
	Labels         *map[string]string
}

func ptr[T any](v T) *T {
	return &v
}

type StartDeploymentRequest struct {
	AdmClient      *restClient.ClientWithResponses
	DpPackageName  string
	DeploymentType string
	RetryDelay     int
	TestName       string
}

func StartDeployment(opts StartDeploymentRequest) (string, int, error) {
	retCode := http.StatusOK
	if DpConfigs[opts.DpPackageName] == nil {
		return "", retCode, fmt.Errorf("deployment package %s not found in configuration", opts.DpPackageName)
	}
	displayName := fmt.Sprintf("%s-%s", opts.DpPackageName, opts.TestName)
	useDP := DpConfigs[opts.DpPackageName].(map[string]any)

	// Check if virt-extension DP is already running, do not recreate a new one
	if opts.DpPackageName == "virt-extension" {
		deployments, retCode, err := getDeploymentPerCluster(opts.AdmClient)
		if err != nil || retCode != 200 {
			return "", retCode, fmt.Errorf("failed to get deployments per cluster: %v, status code: %d", err, retCode)
		}
		for _, deployment := range deployments {
			if *deployment.DeploymentDisplayName == displayName && *deployment.Status.State == "RUNNING" {
				fmt.Printf("%s deployment already exists in cluster %s, skipping creation\n", useDP["deployPackage"], TestClusterID)
				return "", retCode, nil
			}
		}
	}

	err := DeleteAndRetryUntilDeleted(opts.AdmClient, displayName, retryCount, time.Duration(opts.RetryDelay)*time.Second)
	if err != nil {
		return "", retCode, err
	}

	labels := useDP["labels"].(map[string]string)

	overrideValues := useDP["overrideValues"].([]map[string]any)

	deployID, retCode, err := createDeployment(opts.AdmClient, CreateDeploymentParams{
		ClusterID:      useDP["clusterId"].(string),
		DpName:         useDP["deployPackage"].(string),
		AppNames:       useDP["appNames"].([]string),
		AppVersion:     useDP["deployPackageVersion"].(string),
		DisplayName:    displayName,
		ProfileName:    useDP["profileName"].(string),
		DeploymentType: opts.DeploymentType,
		OverrideValues: overrideValues,
		Labels:         &labels,
	})
	if err != nil {
		return "", retCode, err
	}

	fmt.Printf("Created %s deployment successfully, deployment id %s\n", displayName, deployID)

	err = waitForDeploymentStatus(opts.AdmClient, displayName, restClient.RUNNING, retryCount, time.Duration(opts.RetryDelay)*time.Second)
	if err != nil {
		return "", retCode, err
	}
	fmt.Printf("%s deployment is now in RUNNING status\n", displayName)

	return deployID, retCode, nil
}

func DeleteDeployment(client *restClient.ClientWithResponses, deployID string) error {
	resp, err := client.DeploymentServiceDeleteDeploymentWithResponse(context.TODO(), deployID, nil)
	if err != nil || resp.StatusCode() != 200 {
		return fmt.Errorf("failed to delete deployment: %v, status: %d", err, resp.StatusCode())
	}
	return nil
}

func deploymentExists(deployments []restClient.Deployment, displayName string) bool {
	for _, d := range deployments {
		if *d.DisplayName == displayName {
			return true
		}
	}
	return false
}

func getDeploymentPerCluster(client *restClient.ClientWithResponses) ([]restClient.DeploymentInstancesCluster, int, error) {
	resp, err := client.DeploymentServiceListDeploymentsPerClusterWithResponse(context.TODO(), TestClusterID, nil)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return nil, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return nil, resp.StatusCode(), fmt.Errorf("failed to list deployment cluster: %v", string(resp.Body))
	}

	return resp.JSON200.DeploymentInstancesCluster, resp.StatusCode(), nil
}

func getDeployments(client *restClient.ClientWithResponses) ([]restClient.Deployment, int, error) {
	resp, err := client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return nil, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return nil, resp.StatusCode(), fmt.Errorf("failed to list deployments: %v", string(resp.Body))
	}

	return resp.JSON200.Deployments, resp.StatusCode(), nil
}

func GetDeployment(client *restClient.ClientWithResponses, deployID string) (restClient.Deployment, int, error) {
	resp, err := client.DeploymentServiceGetDeploymentWithResponse(context.TODO(), deployID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return restClient.Deployment{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return restClient.Deployment{}, resp.StatusCode(), fmt.Errorf("failed to get deployment: %v", string(resp.Body))
	}

	return resp.JSON200.Deployment, resp.StatusCode(), nil
}

func waitForDeploymentStatus(client *restClient.ClientWithResponses, displayName string, status restClient.DeploymentStatusState, retries int, delay time.Duration) error {
	currState := "UNKNOWN"
	for range retries {
		deployments, retCode, err := getDeployments(client)
		if err != nil || retCode != 200 {
			return fmt.Errorf("failed to get deployments: %v", err)
		}

		for _, d := range deployments {
			// In case there's several deployments only use the one with the same display name
			if *d.DisplayName == displayName {
				currState = string(*d.Status.State)
			}

			if *d.DisplayName == displayName && currState == string(status) {
				fmt.Printf("Waiting for deployment %s state %s ---> %s\n", displayName, currState, status)
				return nil
			}
		}

		fmt.Printf("Waiting for deployment %s state %s ---> %s\n", displayName, currState, status)
		time.Sleep(delay)
	}

	return fmt.Errorf("deployment %s did not reach status %s after %d retries", displayName, status, retries)
}

func FindDeploymentIDByDisplayName(client *restClient.ClientWithResponses, displayName string) string {
	deployments, retCode, err := getDeployments(client)
	if err != nil || retCode != 200 {
		return ""
	}

	for _, d := range deployments {
		if *d.DisplayName == displayName {
			return *d.DeployId
		}
	}

	return ""
}

func deleteDeploymentByDisplayName(client *restClient.ClientWithResponses, displayName string) error {
	if deployID := FindDeploymentIDByDisplayName(client, displayName); deployID != "" {
		err := DeleteDeployment(client, deployID)
		if err != nil {
			return fmt.Errorf("failed to delete deployment %s: %v", displayName, err)
		}
		fmt.Printf("%s deployment deleted\n", displayName)
		return nil
	}

	fmt.Printf("%s deployment not found for deletion\n", displayName)
	return nil
}

func createDeployment(client *restClient.ClientWithResponses, params CreateDeploymentParams) (string, int, error) {
	reqBody := restClient.DeploymentServiceCreateDeploymentJSONRequestBody{
		AppName:        params.DpName,
		AppVersion:     params.AppVersion,
		DeploymentType: ptr(params.DeploymentType),
		DisplayName:    ptr(params.DisplayName),
		ProfileName:    ptr(params.ProfileName),
	}

	var targetClusters []restClient.TargetClusters
	if params.DeploymentType == "targeted" {
		for _, v := range *ptr(params.AppNames) {
			targetClusters = append(targetClusters, restClient.TargetClusters{
				AppName:   ptr(v),
				ClusterId: ptr(params.ClusterID),
			})
		}
	} else if params.DeploymentType == "auto-scaling" {
		for _, v := range *ptr(params.AppNames) {
			targetClusters = append(targetClusters, restClient.TargetClusters{
				AppName: ptr(v),
				Labels:  ptr(*params.Labels),
			})
		}
	}
	reqBody.TargetClusters = &targetClusters

	var overrideValues []restClient.OverrideValues
	for _, v := range *ptr(params.OverrideValues) {
		overrideValues = append(overrideValues, restClient.OverrideValues{
			AppName:         v["appName"].(string),
			TargetNamespace: ptr(v["targetNamespace"].(string)),
			Values: func() *map[string]any {
				convertedMap := make(map[string]any)
				if v["targetValues"] == nil {
					return nil
				}
				maps.Copy(convertedMap, v["targetValues"].(map[string]any))

				return &convertedMap
			}(),
		})
	}

	reqBody.OverrideValues = &overrideValues

	deployID, retCode, err := createDeploymentCmd(client, &reqBody)
	if err != nil {
		return "", retCode, err
	}

	return deployID, retCode, nil
}

func DeleteAndRetryUntilDeleted(client *restClient.ClientWithResponses, displayName string, retries int, delay time.Duration) error {
	// Attempt to delete the deployment
	if err := deleteDeploymentByDisplayName(client, displayName); err != nil {
		return fmt.Errorf("initial deletion failed: %v", err)
	}

	// Retry until the deployment is confirmed deleted
	for range retries {
		deployments, retCode, err := getDeployments(client)
		if err != nil || retCode != 200 {
			return fmt.Errorf("failed to get deployments: %v", err)
		}

		if !deploymentExists(deployments, displayName) {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("deployment %s not deleted after %d retries", displayName, retries)
}

func createDeploymentCmd(admClient *restClient.ClientWithResponses, reqBody *restClient.DeploymentServiceCreateDeploymentJSONRequestBody) (string, int, error) {
	resp, err := admClient.DeploymentServiceCreateDeploymentWithResponse(context.TODO(), *reqBody)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return "", resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return "", resp.StatusCode(), fmt.Errorf("failed to create deployment: %v", string(resp.Body))
	}

	return resp.JSON200.DeploymentId, resp.StatusCode(), nil
}

func GetDeploymentsStatus(admClient *restClient.ClientWithResponses, labels *[]string) (*restClient.GetDeploymentsStatusResponse, int, error) {
	var params *restClient.DeploymentServiceGetDeploymentsStatusParams
	if labels != nil {
		params = &restClient.DeploymentServiceGetDeploymentsStatusParams{
			Labels: labels,
		}
	}
	resp, err := admClient.DeploymentServiceGetDeploymentsStatusWithResponse(context.TODO(), params)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &restClient.GetDeploymentsStatusResponse{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &restClient.GetDeploymentsStatusResponse{}, resp.StatusCode(), fmt.Errorf("failed to get deployment status: %v", string(resp.Body))
	}

	return resp.JSON200, resp.StatusCode(), nil
}
