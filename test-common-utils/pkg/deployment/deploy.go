// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
	"maps"
	"net/http"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

var DpConfigs = map[string]any{
	NginxAppName: map[string]any{
		"appNames":             []string{"nginx"},
		"deployPackage":        "nginx-app",
		"deployPackageVersion": "0.1.0",
		"profileName":          "testing-default",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
	CirrosAppName: map[string]any{
		"appNames":             []string{"cirros-container-disk"},
		"deployPackage":        "cirros-container-disk",
		"deployPackageVersion": "0.1.0",
		"profileName":          "default",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
	VirtualizationExtensionAppName: map[string]any{
		"appNames":             []string{"kubevirt", "cdi", "kube-helper"},
		"deployPackage":        "virtualization",
		"deployPackageVersion": "0.3.7",
		"profileName":          "with-software-emulation-profile-nosm",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
	WordpressAppName: map[string]any{
		"appNames":             []string{"wordpress"},
		"deployPackage":        "wordpress",
		"deployPackageVersion": "0.1.0",
		"profileName":          "testing",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
}

const (
	NginxAppName                   = "nginx"
	CirrosAppName                  = "cirros-container-disk"
	WordpressAppName               = "wordpress"
	VirtualizationExtensionAppName = "virtualization-extension"
)

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
	DeploymentTimeout = 20 * time.Second // 20 seconds

	RetryCount = 20 // Number of retries for deployment operations

	DeleteTimeout = 10 * time.Second // Timeout for deletion operations
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
	AdmClient         *restClient.ClientWithResponses
	DpPackageName     string
	DeploymentType    string
	DeploymentTimeout time.Duration
	DeleteTimeout     time.Duration
	TestName          string
}

func StartDeployment(opts StartDeploymentRequest) (string, int, error) {
	retCode := http.StatusOK
	if DpConfigs[opts.DpPackageName] == nil {
		return "", retCode, fmt.Errorf("deployment package %s not found in configuration", opts.DpPackageName)
	}
	useDP := DpConfigs[opts.DpPackageName].(map[string]any)

	displayName := FormDisplayName(opts.DpPackageName, opts.TestName)
	// Check if virt-extension DP is already running, do not recreate a new one
	if opts.DpPackageName == "virt-extension" {
		deployments, retCode, err := getDeploymentPerCluster(opts.AdmClient)
		if err != nil || retCode != 200 {
			return "", retCode, fmt.Errorf("failed to get deployments per cluster: %v, status code: %d", err, retCode)
		}
		for _, deployment := range deployments {
			if *deployment.DeploymentDisplayName == displayName && *deployment.Status.State == "RUNNING" {
				fmt.Printf("%s deployment already exists in cluster %s, skipping creation\n", useDP["deployPackage"], types.TestClusterID)
				return "", retCode, nil
			}
		}
	}

	err := DeleteAndRetryUntilDeleted(opts.AdmClient, displayName, types.RetryCount, opts.DeleteTimeout)
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

	err = waitForDeploymentStatus(opts.AdmClient, displayName, restClient.RUNNING, types.RetryCount, opts.DeploymentTimeout)
	if err != nil {
		return "", retCode, err
	}
	fmt.Printf("%s deployment is now in RUNNING status\n", displayName)

	return deployID, retCode, nil
}

func DeleteDeployment(client *restClient.ClientWithResponses, deployID string) (int, error) {
	resp, err := client.DeploymentServiceDeleteDeploymentWithResponse(context.TODO(), deployID, nil)
	if err != nil || resp == nil || resp.StatusCode() != http.StatusOK {
		status := 0
		if resp != nil {
			status = resp.StatusCode()
		}
		return status, err
	}
	return resp.StatusCode(), nil
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
	resp, err := client.DeploymentServiceListDeploymentsPerClusterWithResponse(context.TODO(), types.TestClusterID, nil)
	if err != nil || resp == nil || resp.StatusCode() != 200 {
		if err != nil {
			if resp != nil {
				return nil, resp.StatusCode(), fmt.Errorf("%v", err)
			}
			return nil, 0, fmt.Errorf("%v", err)
		}
		if resp != nil {
			return nil, resp.StatusCode(), fmt.Errorf("failed to list deployment cluster: %v", string(resp.Body))
		}
		return nil, 0, fmt.Errorf("failed to list deployment cluster: response is nil")
	}

	return resp.JSON200.DeploymentInstancesCluster, resp.StatusCode(), nil
}

func getDeployments(client *restClient.ClientWithResponses) ([]restClient.Deployment, int, error) {
	resp, err := client.DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
	if err != nil || resp == nil || resp.StatusCode() != 200 {
		if err != nil {
			if resp != nil {
				return nil, resp.StatusCode(), fmt.Errorf("%v", err)
			}
			return nil, 0, fmt.Errorf("%v", err)
		}
		if resp != nil {
			return nil, resp.StatusCode(), fmt.Errorf("failed to list deployments: %v", string(resp.Body))
		}
		return nil, 0, fmt.Errorf("failed to list deployments: response is nil")
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
	for i := 0; i < retries; i++ {
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
		_, err := DeleteDeployment(client, deployID)
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
	for i := 0; i < retries; i++ {
		fmt.Printf("Checking if deployment %s is deleted (%d/%d)\n", displayName, i+1, retries)
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

func FormDisplayName(dpPackageName, testName string) string {
	return fmt.Sprintf("%s-%s", dpPackageName, testName)
}

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
	DpConfigs = CopyOriginalDpConfig(originalDpConfigs)

	if dpConfig, ok := DpConfigs[dpConfigName].(map[string]any); ok {
		dpConfig[key] = value
		DpConfigs[dpConfigName] = dpConfig
	} else {
		return fmt.Errorf("failed to assert type of deploy.DpConfigs[%s] as map[string]any", dpConfigName)
	}
	return nil
}

func GetDeployApps(client *restClient.ClientWithResponses, deployID string) ([]*restClient.App, error) {
	deployments, retCode, err := getDeploymentPerCluster(client)
	if err != nil || retCode != 200 {
		return []*restClient.App{}, fmt.Errorf("failed to get deployments: %v", err)
	}

	for _, d := range deployments {
		if *d.DeploymentUid == deployID {
			apps := make([]*restClient.App, len(*d.Apps))
			for i, app := range *d.Apps {
				apps[i] = &app
			}
			return apps, nil
		}
	}

	return []*restClient.App{}, fmt.Errorf("did not find deployment id %s", deployID)
}
