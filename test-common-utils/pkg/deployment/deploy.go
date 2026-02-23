// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	deploymentv1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
)

var DpConfigs = map[string]any{
	NginxAppName: map[string]any{
		"appNames":             []string{"nginx"},
		"deployPackage":        "nginx-app",
		"deployPackageVersion": "0.1.0",
		"profileName":          "testing-default",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"user": "customer"},
		"overrideValues":       []map[string]any{},
	},
	CirrosAppName: map[string]any{
		"appNames":             []string{"cirros-container-disk"},
		"deployPackage":        "cirros-container-disk",
		"deployPackageVersion": "0.1.0",
		"profileName":          "default",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"user": "customer"},
		"overrideValues":       []map[string]any{},
	},
	VirtualizationExtensionAppName: map[string]any{
		"appNames":             []string{"kubevirt", "cdi", "kube-helper"},
		"deployPackage":        "virtualization",
		"deployPackageVersion": "0.5.1",
		"profileName":          "with-software-emulation-profile-nosm",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"user": "customer"},
		"overrideValues":       []map[string]any{},
	},
	WordpressAppName: map[string]any{
		"appNames":             []string{"wordpress"},
		"deployPackage":        "wordpress",
		"deployPackageVersion": "0.1.1",
		"profileName":          "testing",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"user": "customer"},
		"overrideValues":       []map[string]any{},
	},
	HttpbinAppName: map[string]any{
		"appNames":             []string{"httpbin"},
		"deployPackage":        "httpbin",
		"deployPackageVersion": "2.3.5",
		"profileName":          "default",
		"clusterId":            types.TestClusterID,
		"labels":               map[string]string{"user": "customer"},
		"overrideValues":       []map[string]any{},
	},
}

const (
	NginxAppName                   = "nginx"
	CirrosAppName                  = "cirros-container-disk"
	WordpressAppName               = "wordpress"
	HttpbinAppName                 = "httpbin"
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
	// Increased from 20s to 40s to allow more time for new clusters to schedule pods
	DeploymentTimeout = 40 * time.Second // 40 seconds

	// RetryCount is the number of retries for deployment operations
	// Increased from 20 to 30 to allow more time for new clusters
	RetryCount = 30 // Number of retries for deployment operations

	DeleteTimeout = 15 * time.Second // Timeout for deletion operations
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
	ReuseFlag         bool
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
			if *deployment.DisplayName == displayName && *deployment.Status.State == restClient.RUNNING {
				fmt.Printf("%s deployment already exists in cluster %s, skipping creation\n", useDP["deployPackage"], types.GetTestClusterID())
				return "", retCode, nil
			}
		}
	}

	// Enable if you want to resue existing deployed app
	if opts.ReuseFlag {
		if deployID := FindDeploymentIDByDisplayName(opts.AdmClient, displayName); deployID != "" {
			fmt.Printf("Deployment exists. use it")
			_, err := GetDeployApps(opts.AdmClient, deployID)
			if err != nil {
				return "", retCode, err
			}
			return deployID, retCode, nil
		}
	}

	err := DeleteAndRetryUntilDeleted(opts.AdmClient, displayName, types.RetryCount, opts.DeleteTimeout)
	if err != nil {
		return "", retCode, err
	}

	// Wait for cluster to be ready before creating deployment
	// This is especially important for newly created clusters
	if waitErr := WaitForClusterReady(opts.AdmClient); waitErr != nil {
		fmt.Printf("Warning: cluster readiness check failed: %v, continuing anyway\n", waitErr)
	}

	// For auto-scaling deployments, update labels with actual cluster labels
	if opts.DeploymentType == DeploymentTypeAutoScaling {
		if updateErr := UpdateDpConfigsWithClusterLabels(opts.AdmClient); updateErr != nil {
			fmt.Printf("Warning: failed to update DpConfigs with cluster labels: %v\n", updateErr)
			// Continue anyway, the default labels might work
		}
		// Re-read the config after potential update
		useDP = DpConfigs[opts.DpPackageName].(map[string]any)
	}

	labels := useDP["labels"].(map[string]string)

	overrideValues := useDP["overrideValues"].([]map[string]any)

	// Get cluster ID from API for targeted deployments
	clusterID := ""
	if opts.DeploymentType == DeploymentTypeTargeted {
		clusterID, err = GetFirstClusterID(opts.AdmClient)
		if err != nil {
			fmt.Printf("Warning: failed to get cluster ID from API: %v\n", err)
			// Try kubectl fallback
			clusterID, err = GetClusterIDFromKubectl()
			if err != nil {
				fmt.Printf("Warning: kubectl fallback also failed: %v, using env var\n", err)
				clusterID = types.GetTestClusterID()
			}
		}
	}
	fmt.Printf("Using cluster ID: %s (type: %s)\n", clusterID, opts.DeploymentType)

	deployID, retCode, err := createDeployment(opts.AdmClient, CreateDeploymentParams{
		ClusterID:      clusterID,
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

	err = waitForDeploymentStatus(opts.AdmClient, displayName, deploymentv1.State_RUNNING, types.RetryCount, opts.DeploymentTimeout)
	if err != nil {
		return "", retCode, err
	}
	fmt.Printf("%s deployment is now in RUNNING status\n", displayName)

	return deployID, retCode, nil
}

func DeleteDeploymentWithDeleteType(client *restClient.ClientWithResponses, deployID string, deleteType deploymentv1.DeleteType) (int, error) {
	// Convert protobuf enum to REST client enum
	var restDeleteType restClient.DeploymentV1DeleteType
	switch deleteType {
	case deploymentv1.DeleteType_ALL:
		restDeleteType = restClient.ALL
	case deploymentv1.DeleteType_PARENT_ONLY:
		restDeleteType = restClient.PARENTONLY
	default:
		restDeleteType = restClient.ALL
	}

	resp, err := client.DeploymentV1DeploymentServiceDeleteDeploymentWithResponse(context.TODO(), deployID, &restClient.DeploymentV1DeploymentServiceDeleteDeploymentParams{
		DeleteType: restDeleteType,
	})
	if err != nil || resp == nil || resp.StatusCode() != http.StatusOK {
		status := 0
		if resp != nil {
			status = resp.StatusCode()
		}
		return status, err
	}
	return resp.StatusCode(), nil
}

func DeleteDeployment(client *restClient.ClientWithResponses, deployID string) (int, error) {
	resp, err := client.DeploymentV1DeploymentServiceDeleteDeploymentWithResponse(context.TODO(), deployID, nil)
	if err != nil || resp == nil || resp.StatusCode() != http.StatusOK {
		status := 0
		if resp != nil {
			status = resp.StatusCode()
		}
		return status, err
	}
	return resp.StatusCode(), nil
}

func deploymentExists(deployments []restClient.DeploymentV1Deployment, displayName string) bool {
	for _, d := range deployments {
		if *d.DisplayName == displayName {
			return true
		}
	}
	return false
}

func getDeploymentPerCluster(admClient *restClient.ClientWithResponses) ([]restClient.DeploymentV1Deployment, int, error) {
	// Use ListDeployments to get all deployments, since ListDeploymentsPerCluster returns different structure
	resp, err := admClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
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

func getDeployments(client *restClient.ClientWithResponses) ([]restClient.DeploymentV1Deployment, int, error) {
	resp, err := client.DeploymentV1DeploymentServiceListDeploymentsWithResponse(context.TODO(), nil)
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

func GetDeployment(client *restClient.ClientWithResponses, deployID string) (restClient.DeploymentV1Deployment, int, error) {
	resp, err := client.DeploymentV1DeploymentServiceGetDeploymentWithResponse(context.TODO(), deployID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return restClient.DeploymentV1Deployment{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return restClient.DeploymentV1Deployment{}, resp.StatusCode(), err
	}

	return resp.JSON200.Deployment, resp.StatusCode(), nil
}

func waitForDeploymentStatus(client *restClient.ClientWithResponses, displayName string, status deploymentv1.State, retries int, delay time.Duration) error {
	currState := "UNKNOWN"

	// Convert protobuf enum to REST client enum
	var targetState restClient.DeploymentV1State
	switch status {
	case deploymentv1.State_RUNNING:
		targetState = restClient.RUNNING
	case deploymentv1.State_DEPLOYING:
		targetState = restClient.DEPLOYING
	case deploymentv1.State_ERROR:
		targetState = restClient.ERROR
	case deploymentv1.State_TERMINATING:
		targetState = restClient.TERMINATING
	case deploymentv1.State_DOWN:
		targetState = restClient.DOWN
	case deploymentv1.State_UPDATING:
		targetState = restClient.UPDATING
	default:
		targetState = restClient.UNKNOWN
	}
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

			if *d.DisplayName == displayName && *d.Status.State == targetState {
				fmt.Printf("Waiting for deployment %s state %s ---> %s\n", displayName, currState, status.String())
				return nil
			}
		}

		fmt.Printf("Waiting for deployment %s state %s ---> %s\n", displayName, currState, status.String())
		time.Sleep(delay)
	}

	return fmt.Errorf("deployment %s did not reach status %s after %d retries", displayName, status.String(), retries)
}

func UpdateDeployment(client *restClient.ClientWithResponses, deployID string, params restClient.DeploymentV1DeploymentServiceUpdateDeploymentJSONRequestBody) (int, error) {
	resp, err := client.DeploymentV1DeploymentServiceUpdateDeploymentWithResponse(context.TODO(), deployID, params)
	if err != nil || resp == nil || resp.StatusCode() != 200 {
		if err != nil {
			if resp != nil {
				return resp.StatusCode(), fmt.Errorf("%v", err)
			}
			return 0, fmt.Errorf("%v", err)
		}
		if resp != nil {
			return resp.StatusCode(), fmt.Errorf("failed to update deployment: %v", string(resp.Body))
		}
		return 0, fmt.Errorf("failed to update deployment: response is nil")
	}

	return resp.StatusCode(), nil
}

func FindDeploymentIDByDisplayName(client *restClient.ClientWithResponses, displayName string) string {
	deployments, retCode, err := getDeployments(client)
	if err != nil || retCode != 200 {
		return ""
	}

	fmt.Printf("FindDeploymentIDByDisplayName displayName 1 : %s\n", displayName)
	for _, d := range deployments {
		fmt.Printf("FindDeploymentIDByDisplayName displayName 2 : %s\n", *d.DisplayName)
		if *d.DisplayName == displayName {
			fmt.Printf("FindDeploymentIDByDisplayName deployID : %s\n", *d.DeployId)
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
	reqBody := restClient.DeploymentV1DeploymentServiceCreateDeploymentJSONRequestBody{
		AppName:        params.DpName,
		AppVersion:     params.AppVersion,
		DeploymentType: ptr(params.DeploymentType),
		DisplayName:    ptr(params.DisplayName),
		ProfileName:    ptr(params.ProfileName),
	}

	var targetClusters []restClient.DeploymentV1TargetClusters
	if params.DeploymentType == "targeted" {
		for _, v := range *ptr(params.AppNames) {
			targetClusters = append(targetClusters, restClient.DeploymentV1TargetClusters{
				AppName:   ptr(v),
				ClusterId: ptr(params.ClusterID),
			})
		}
	} else if params.DeploymentType == "auto-scaling" {
		for _, v := range *ptr(params.AppNames) {
			labels := make(map[string]string)
			if params.Labels != nil {
				for k, v := range *params.Labels {
					labels[k] = v
				}
			}
			targetClusters = append(targetClusters, restClient.DeploymentV1TargetClusters{
				AppName: ptr(v),
				Labels:  &labels,
			})
		}
	}
	reqBody.TargetClusters = &targetClusters

	var overrideValues []restClient.DeploymentV1OverrideValues
	for _, v := range *ptr(params.OverrideValues) {
		overrideValues = append(overrideValues, restClient.DeploymentV1OverrideValues{
			AppName:         v["appName"].(string),
			TargetNamespace: ptr(v["targetNamespace"].(string)),
			Values: func() *restClient.GoogleProtobufStruct {
				if v["targetValues"] == nil {
					return nil
				}
				structMap := make(restClient.GoogleProtobufStruct)
				for key, value := range v["targetValues"].(map[string]any) {
					structMap[key] = &restClient.GoogleProtobufValue{
						"value": value,
					}
				}
				return &structMap
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

func createDeploymentCmd(admClient *restClient.ClientWithResponses, reqBody *restClient.DeploymentV1DeploymentServiceCreateDeploymentJSONRequestBody) (string, int, error) {
	resp, err := admClient.DeploymentV1DeploymentServiceCreateDeploymentWithResponse(context.TODO(), *reqBody)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return "", resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return "", resp.StatusCode(), fmt.Errorf("failed to create deployment: %v", string(resp.Body))
	}

	return resp.JSON200.DeploymentId, resp.StatusCode(), nil
}

func GetDeploymentsStatus(admClient *restClient.ClientWithResponses, labels *[]string) (*restClient.DeploymentV1GetDeploymentsStatusResponse, int, error) {
	var params *restClient.DeploymentV1DeploymentServiceGetDeploymentsStatusParams
	if labels != nil {
		params = &restClient.DeploymentV1DeploymentServiceGetDeploymentsStatusParams{
			Labels: labels,
		}
	}
	resp, err := admClient.DeploymentV1DeploymentServiceGetDeploymentsStatusWithResponse(context.TODO(), params)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &restClient.DeploymentV1GetDeploymentsStatusResponse{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &restClient.DeploymentV1GetDeploymentsStatusResponse{}, resp.StatusCode(), fmt.Errorf("failed to get deployment status: %v", string(resp.Body))
	}

	return resp.JSON200, resp.StatusCode(), nil
}

func FormDisplayName(dpPackageName, testName string) string {
	return fmt.Sprintf("%s-%s", dpPackageName, testName)
}

func DeploymentsList(admClient *restClient.ClientWithResponses) (*[]restClient.DeploymentV1Deployment, int, error) {
	return DeploymentsListWithParams(admClient, nil)
}

func DeploymentsListWithParams(admClient *restClient.ClientWithResponses, params *restClient.DeploymentV1DeploymentServiceListDeploymentsParams) (*[]restClient.DeploymentV1Deployment, int, error) {
	resp, err := admClient.DeploymentV1DeploymentServiceListDeploymentsWithResponse(context.TODO(), params)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.DeploymentV1Deployment{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.DeploymentV1Deployment{}, resp.StatusCode(), err
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

func GetDeployApps(client *restClient.ClientWithResponses, deployID string) ([]*restClient.DeploymentV1App, error) {
	deployments, retCode, err := getDeploymentPerCluster(client)
	if err != nil || retCode != 200 {
		return []*restClient.DeploymentV1App{}, fmt.Errorf("failed to get deployments: %v", err)
	}

	for _, d := range deployments {
		if *d.DeployId == deployID {
			apps := make([]*restClient.DeploymentV1App, len(*d.Apps))
			for i, app := range *d.Apps {
				apps[i] = &app
				fmt.Println("app.Name : ", *app.Name)
			}
			return apps, nil
		}
	}

	return []*restClient.DeploymentV1App{}, fmt.Errorf("did not find deployment id %s", deployID)
}

// GetFirstClusterID retrieves the first available cluster ID from the deployment API.
// This returns the actual cluster ID (UUID) that should be used for targeted deployments.
// It retries a few times in case clusters are not immediately available.
func GetFirstClusterID(client *restClient.ClientWithResponses) (string, error) {
	maxRetries := 5
	retryDelay := 10 * time.Second

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			fmt.Printf("Retrying GetFirstClusterID (%d/%d)...\n", retry+1, maxRetries)
			time.Sleep(retryDelay)
		}

		resp, err := client.DeploymentV1ClusterServiceListClustersWithResponse(context.TODO(), nil)
		if err != nil {
			fmt.Printf("ListClusters API error: %v\n", err)
			continue
		}

		fmt.Printf("ListClusters API response - Status: %d, Body length: %d\n", resp.StatusCode(), len(resp.Body))

		if resp.StatusCode() != 200 {
			fmt.Printf("ListClusters non-200 response: %s\n", string(resp.Body))
			continue
		}

		if resp.JSON200 == nil {
			fmt.Printf("ListClusters JSON200 is nil\n")
			continue
		}

		fmt.Printf("ListClusters returned %d clusters\n", len(resp.JSON200.Clusters))

		if len(resp.JSON200.Clusters) == 0 {
			continue
		}

		// Found clusters, get the first one
		clusterID := ""
		clusterName := ""
		if resp.JSON200.Clusters[0].Id != nil {
			clusterID = *resp.JSON200.Clusters[0].Id
		}
		if resp.JSON200.Clusters[0].Name != nil {
			clusterName = *resp.JSON200.Clusters[0].Name
		}

		fmt.Printf("Found cluster - Name: %s, ID: %s\n", clusterName, clusterID)
		return clusterID, nil
	}

	return "", fmt.Errorf("no clusters found after %d retries", maxRetries)
}

// WaitForClusterReady waits for the first cluster to be available for deployments.
// A cluster is considered ready when it exists and has an ID assigned.
// This should be called before creating deployments on a newly created cluster.
func WaitForClusterReady(client *restClient.ClientWithResponses) error {
	maxRetries := 15
	retryDelay := 20 * time.Second // Wait 20 seconds between checks

	fmt.Println("Waiting for cluster to be ready for deployments...")

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			time.Sleep(retryDelay)
			fmt.Printf("Checking cluster readiness (%d/%d)...\n", retry+1, maxRetries)
		}

		resp, err := client.DeploymentV1ClusterServiceListClustersWithResponse(context.TODO(), nil)
		if err != nil {
			fmt.Printf("ListClusters API error while checking readiness: %v\n", err)
			continue
		}

		if resp.StatusCode() != 200 || resp.JSON200 == nil || len(resp.JSON200.Clusters) == 0 {
			fmt.Println("No clusters available yet, waiting...")
			continue
		}

		cluster := resp.JSON200.Clusters[0]
		clusterName := ""
		if cluster.Name != nil {
			clusterName = *cluster.Name
		}

		// Check if cluster exists and has an ID assigned
		// Having an ID means the cluster has been created and registered
		if cluster.Id != nil && *cluster.Id != "" {
			fmt.Printf("Cluster %s is ready (ID: %s)\n", clusterName, *cluster.Id)
			// Wait a bit more to ensure the cluster is fully initialized
			time.Sleep(10 * time.Second)
			return nil
		}

		fmt.Printf("Cluster %s exists but no ID yet, waiting...\n", clusterName)
	}

	// After all retries, if we have a cluster, proceed anyway
	// The deployment might work or provide a more meaningful error
	fmt.Println("Warning: Cluster readiness check timed out, proceeding anyway...")
	return nil
}

// GetFirstClusterLabels retrieves the labels of the first available cluster from the deployment API.
// This is useful for auto-scaling deployments that need to match cluster labels.
func GetFirstClusterLabels(client *restClient.ClientWithResponses) (map[string]string, error) {
	maxRetries := 10
	retryDelay := 5 * time.Second

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			time.Sleep(retryDelay)
			fmt.Printf("Retrying GetFirstClusterLabels (%d/%d)...\n", retry+1, maxRetries)
		}

		resp, err := client.DeploymentV1ClusterServiceListClustersWithResponse(context.TODO(), nil)
		if err != nil {
			fmt.Printf("ListClusters API error for labels: %v\n", err)
			continue
		}

		if resp.StatusCode() != 200 || resp.JSON200 == nil || len(resp.JSON200.Clusters) == 0 {
			continue
		}

		// Found clusters, get the first one's labels
		cluster := resp.JSON200.Clusters[0]
		if cluster.Labels != nil && len(*cluster.Labels) > 0 {
			fmt.Printf("Found cluster labels: %v\n", *cluster.Labels)
			return *cluster.Labels, nil
		}

		// Cluster has no labels, return empty map
		fmt.Printf("Cluster has no labels\n")
		return map[string]string{}, nil
	}

	return nil, fmt.Errorf("no clusters found after %d retries", maxRetries)
}

// UpdateDpConfigsWithClusterLabels updates all DpConfigs entries with the actual cluster labels.
// This ensures auto-scaling deployments can match the cluster.
func UpdateDpConfigsWithClusterLabels(client *restClient.ClientWithResponses) error {
	labels, err := GetFirstClusterLabels(client)
	if err != nil {
		return fmt.Errorf("failed to get cluster labels: %w", err)
	}

	// If cluster has no labels, we can't use auto-scaling deployments effectively
	// In this case, we log a warning but continue (tests using targeted will still work)
	if len(labels) == 0 {
		fmt.Println("Warning: Cluster has no labels. Auto-scaling deployments may not find matching clusters.")
		// Use a default label that might work
		labels = map[string]string{"user": "customer"}
	}

	// Update all DpConfigs with the actual cluster labels
	for dpName, config := range DpConfigs {
		if dpConfig, ok := config.(map[string]any); ok {
			dpConfig["labels"] = labels
			DpConfigs[dpName] = dpConfig
		}
	}

	fmt.Printf("Updated DpConfigs with cluster labels: %v\n", labels)
	return nil
}

// GetClusterIDFromKubectl retrieves the cluster ID using kubectl as a fallback.
// This queries the nexus CRD directly to get the cluster UID.
func GetClusterIDFromKubectl() (string, error) {
	// First, get the list of cluster names
	// #nosec G204 -- Arguments are controlled within application context.
	cmd := exec.Command("kubectl", "get", "clusterinfos.nexus.api.resourcemanager.orchestrator.apis",
		"-n", "nexus", "-o", "jsonpath={.items[0].metadata.uid}")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative CRD names
		// #nosec G204 -- Arguments are controlled within application context.
		cmd = exec.Command("kubectl", "get", "clusters.cluster.orchestrator.apis",
			"-A", "-o", "jsonpath={.items[0].metadata.uid}")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get cluster ID via kubectl: %w", err)
		}
	}

	clusterID := strings.TrimSpace(string(output))
	if clusterID == "" {
		return "", fmt.Errorf("kubectl returned empty cluster ID")
	}

	fmt.Printf("Got cluster ID from kubectl: %s\n", clusterID)
	return clusterID, nil
}
