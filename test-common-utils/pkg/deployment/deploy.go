// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"
	"net/http"
	"os"
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
	// Set to 20s for faster polling
	DeploymentTimeout = 20 * time.Second // 20 seconds

	// RetryCount is the number of retries for deployment operations
	// Set to 15 to allow up to 5 minutes for deployments to become ready per attempt
	RetryCount = 15 // Number of retries for deployment operations

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

	// Add a small delay before creating deployment to reduce cluster resource pressure
	// This helps when multiple tests are running concurrently
	time.Sleep(2 * time.Second)

	// Retry deployment creation up to 2 times on ERROR state
	maxRetries := 2
	var deployID string
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		deployID, retCode, err = createDeployment(opts.AdmClient, CreateDeploymentParams{
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

		fmt.Printf("Created %s deployment successfully, deployment id %s (attempt %d/%d)\n", displayName, deployID, attempt, maxRetries)

		lastErr = waitForDeploymentStatus(opts.AdmClient, displayName, deploymentv1.State_RUNNING, types.RetryCount, opts.DeploymentTimeout)
		if lastErr == nil {
			fmt.Printf("%s deployment is now in RUNNING status\n", displayName)
			return deployID, retCode, nil
		}

		// Check if it's an ERROR state that might be recoverable
		if strings.Contains(lastErr.Error(), "is in ERROR state") && attempt < maxRetries {
			fmt.Printf("Deployment %s failed with ERROR state, cleaning up and retrying (attempt %d/%d)\n", displayName, attempt, maxRetries)
			// Delete the failed deployment
			if deployID != "" {
				_, _ = DeleteDeployment(opts.AdmClient, deployID)
				time.Sleep(10 * time.Second) // Wait for cleanup
			}
			// Delete by display name to ensure cleanup
			_ = DeleteAndRetryUntilDeleted(opts.AdmClient, displayName, 10, opts.DeleteTimeout)
			continue
		}

		// Non-recoverable error or max retries reached
		break
	}

	if lastErr != nil {
		FetchAndPrintClusterState(opts.AdmClient, clusterID)
	}

	return "", retCode, lastErr
}

func FetchAndPrintClusterState(client *restClient.ClientWithResponses, clusterID string) {
	if clusterID == "" {
		var err error
		clusterID, err = GetFirstClusterID(client)
		if err != nil || clusterID == "" {
			fmt.Printf("Deployment failed, and couldn't determine cluster ID to fetch kubeconfig: %v\n", err)
			return
		}
	}

	fmt.Printf("Deployment failed. Fetching kubeconfig for cluster %s to dump cluster state...\n", clusterID)
	reqBody := restClient.DeploymentV1ClusterServiceGetKubeConfigJSONRequestBody{
		ClusterId: clusterID,
	}
	params := &restClient.DeploymentV1ClusterServiceGetKubeConfigParams{
		ConnectProtocolVersion: 1,
	}

	resp, err := client.DeploymentV1ClusterServiceGetKubeConfigWithResponse(context.TODO(), params, reqBody)
	if err != nil {
		fmt.Printf("Failed to fetch kubeconfig API call: %v\n", err)
		return
	}

	if resp.StatusCode() != 200 {
		fmt.Printf("Failed to fetch kubeconfig, status: %d\n", resp.StatusCode())
		return
	}

	if resp.JSON200 == nil || resp.JSON200.KubeConfigInfo == nil || resp.JSON200.KubeConfigInfo.KubeConfig == nil {
		fmt.Printf("Failed to fetch kubeconfig, no kubeconfig data returned\n")
		return
	}

	kubeConfigData := *resp.JSON200.KubeConfigInfo.KubeConfig

	tmpFile, err := os.CreateTemp("", "failed-cluster-kubeconfig-*")
	if err != nil {
		fmt.Printf("Failed to create temp file for kubeconfig: %v\n", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	if err := os.WriteFile(tmpFile.Name(), kubeConfigData, 0600); err != nil {
		fmt.Printf("Failed to write kubeconfig temp file: %v\n", err)
		return
	}
	fmt.Printf("Successfully fetched kubeconfig, running 'kubectl get all -A'...\n")

	cmd := exec.Command("kubectl", "--kubeconfig", tmpFile.Name(), "get", "all", "-A")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running kubectl: %v\n", err)
	}
	fmt.Printf("Cluster State:\n%s\n", string(output))
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

		var statusMessage string
		for _, d := range deployments {
			// In case there's several deployments only use the one with the same display name
			if *d.DisplayName == displayName {
				fmt.Printf("Deployment Details:\n  ID: %s\n  Display Name: %s\n  State: %s\n  Message: %s\n  Apps: %v\n",
					*d.DeployId, *d.DisplayName, *d.Status.State, statusMessage, d.Apps)
				if d.Status.Summary != nil {
					var summaryParts []string
					if d.Status.Summary.Total != nil {
						summaryParts = append(summaryParts, fmt.Sprintf("Total: %d", *d.Status.Summary.Total))
					}
					if d.Status.Summary.Running != nil {
						summaryParts = append(summaryParts, fmt.Sprintf("Running: %d", *d.Status.Summary.Running))
					}
					if d.Status.Summary.Down != nil {
						summaryParts = append(summaryParts, fmt.Sprintf("Down: %d", *d.Status.Summary.Down))
					}
					if d.Status.Summary.Unknown != nil {
						summaryParts = append(summaryParts, fmt.Sprintf("Unknown: %d", *d.Status.Summary.Unknown))
					}
					if d.Status.Summary.Type != nil {
						summaryParts = append(summaryParts, fmt.Sprintf("Type: %s", *d.Status.Summary.Type))
					}
					if len(summaryParts) > 0 {
						fmt.Printf("  Summary: %s\n", strings.Join(summaryParts, ", "))
					} else {
						fmt.Printf("  Summary: empty\n")
					}
				}
				currState = string(*d.Status.State)
				// Get status message for better diagnostics
				if d.Status.Message != nil {
					statusMessage = *d.Status.Message
				}

				// Check if deployment is in ERROR state - fail fast instead of waiting
				if *d.Status.State == restClient.ERROR {
					return fmt.Errorf("deployment %s is in ERROR state: %s", displayName, statusMessage)
				}

				// Check if deployment has NO_TARGET_CLUSTERS - this means the cluster isn't matching
				// This is a configuration issue, not a transient error, so fail fast
				if *d.Status.State == restClient.NOTARGETCLUSTERS {
					return fmt.Errorf("deployment %s has NO_TARGET_CLUSTERS: cluster not found or labels don't match. Message: %s", displayName, statusMessage)
				}
			}

			if *d.DisplayName == displayName && *d.Status.State == targetState {
				fmt.Printf("Deployment %s reached target state %s\n", displayName, status.String())
				return nil
			}
		}

		fmt.Printf("Waiting for deployment %s state %s ---> %s (attempt %d/%d)%s\n",
			displayName, currState, status.String(), i+1, retries,
			func() string {
				if statusMessage != "" {
					return fmt.Sprintf(" [%s]", statusMessage)
				}
				return ""
			}())
		time.Sleep(delay)
	}

	return fmt.Errorf("deployment %s did not reach status %s after %d retries (last state: %s)", displayName, status.String(), retries, currState)
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

// UpdateDpConfigsWithClusterLabels updates all DpConfigs entries with valid cluster labels.
// This ensures auto-scaling deployments can match the cluster.
// Only labels that pass validation are used (max 40 chars, matching pattern, non-empty values).
func UpdateDpConfigsWithClusterLabels(client *restClient.ClientWithResponses) error {
	labels, err := GetFirstClusterLabels(client)
	if err != nil {
		return fmt.Errorf("failed to get cluster labels: %w", err)
	}

	// Filter labels to only include valid ones for deployment matching
	// Validation rules:
	// - Key must not contain uppercase letters or special chars that aren't allowed
	// - Value must be 1-40 chars and match pattern: ^[a-z0-9]([-_.=,a-z0-9/]{0,38}[a-z0-9])?$
	validLabels := filterValidLabels(labels)

	// If no valid labels found, use default
	if len(validLabels) == 0 {
		fmt.Println("Warning: No valid cluster labels found. Using default labels.")
		validLabels = map[string]string{"user": "customer"}
	}

	fmt.Printf("Using filtered valid labels for deployments: %v\n", validLabels)

	// Update all DpConfigs with the valid cluster labels
	for dpName, config := range DpConfigs {
		if dpConfig, ok := config.(map[string]any); ok {
			dpConfig["labels"] = validLabels
			DpConfigs[dpName] = dpConfig
		}
	}

	fmt.Printf("Updated DpConfigs with cluster labels: %v\n", validLabels)
	return nil
}

// filterValidLabels filters cluster labels to only include ones valid for deployment matching.
// Valid labels must have:
// - Non-empty values (at least 1 character)
// - Values that are 40 chars or less
// - Values matching pattern: lowercase alphanumeric, can contain -_.=,/
// Additionally, we prioritize known good labels like "user" that are commonly used for matching.
func filterValidLabels(labels map[string]string) map[string]string {
	validLabels := make(map[string]string)

	// Priority labels that are commonly used for auto-scaling matching
	priorityKeys := []string{"user", "environment", "env", "tier", "app", "component"}

	// First, check for priority labels
	for _, key := range priorityKeys {
		if value, exists := labels[key]; exists && isValidLabelValue(value) {
			validLabels[key] = value
		}
	}

	// If we found priority labels, use them
	if len(validLabels) > 0 {
		return validLabels
	}

	// Otherwise, filter all labels for valid ones
	for key, value := range labels {
		if isValidLabelValue(value) && isValidLabelKey(key) {
			validLabels[key] = value
		}
	}

	return validLabels
}

// isValidLabelValue checks if a label value meets the validation requirements.
// - Must be 1-40 characters
// - Must match pattern: lowercase alphanumeric with some special chars
func isValidLabelValue(value string) bool {
	if len(value) == 0 || len(value) > 40 {
		return false
	}

	// Simple validation: must start and end with alphanumeric, can contain -_.=,/
	// Pattern: ^[a-z0-9]([-_.=,a-z0-9/]{0,38}[a-z0-9])?$
	if len(value) == 1 {
		return isLowerAlphanumeric(rune(value[0]))
	}

	if !isLowerAlphanumeric(rune(value[0])) || !isLowerAlphanumeric(rune(value[len(value)-1])) {
		return false
	}

	for _, c := range value[1 : len(value)-1] {
		if !isLowerAlphanumeric(c) && c != '-' && c != '_' && c != '.' && c != '=' && c != ',' && c != '/' {
			return false
		}
	}

	return true
}

// isValidLabelKey checks if a label key is simple enough for deployment matching.
// Avoids complex keys with multiple slashes or special prefixes.
func isValidLabelKey(key string) bool {
	if len(key) == 0 || len(key) > 63 {
		return false
	}

	// Avoid keys with complex prefixes (e.g., kubernetes.io/, x-k8s.io/)
	// These are usually system labels not meant for deployment matching
	if strings.Contains(key, "kubernetes.io") ||
		strings.Contains(key, "k8s.io") ||
		strings.Contains(key, "orchestrator.intel.com") {
		return false
	}

	return true
}

// isLowerAlphanumeric checks if a character is lowercase letter or digit.
func isLowerAlphanumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
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
