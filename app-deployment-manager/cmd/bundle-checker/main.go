// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	fmt.Printf("App-Orch Bundle Checker\n\n")
	fmt.Printf("This tool uses the kubernetes API (set your KUBECONFIG environment variable appropriately!).\n")
	fmt.Printf("It will check all Fleet Bundles in the cluster and list those that have an error condition.\n")
	fmt.Printf("\n")

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to determine home directory: %s\n", err)
			os.Exit(1)
		}
		kubeconfig = homeDir + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fail to build the k8s config. Error - %s\n", err)
		os.Exit(1)
	}

	dynamicClientSet, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fail to create the dynamic client set. Error - %s\n", err)
		os.Exit(1)
	}

	projectNameMap, err := GetProjectNames(context.Background(), dynamicClientSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get project names: %s\n", err)
		os.Exit(1)
	}

	deploymentNameMap, err := GetDeploymentNames(context.Background(), dynamicClientSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get deployment names: %s\n", err)
		os.Exit(1)
	}

	allBundles, err := ListBundles(context.Background(), dynamicClientSet, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list bundles: %s\n", err)
		os.Exit(1)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Project", "Cluster", "Deployment", "App", "Type", "Reason", "Namespace", "Bundle Name"})
	for _, bundle := range allBundles {
		name := bundle.GetName()

		conditions, found, _ := unstructured.NestedSlice(bundle.Object, "status", "conditions")
		if !found {
			continue
		}

		projectName := "unknown"
		projectID, found, _ := unstructured.NestedString(bundle.Object, "metadata", "labels", "app.edge-orchestrator.intel.com/project-id")
		if found {
			var ok bool
			projectName, ok = projectNameMap[projectID]
			if !ok {
				projectName = projectID
			}
		}

		clusterName, found, _ := unstructured.NestedString(bundle.Object, "metadata", "labels", "fleet.cattle.io/cluster")
		if !found {
			clusterName = "unknown"
		}

		appName, found, _ := unstructured.NestedString(bundle.Object, "metadata", "labels", "app.edge-orchestrator.intel.com/app-name")
		if !found {
			appName = "unknown"
		}

		deploymentName := "unknown"
		deploymentID, found, _ := unstructured.NestedString(bundle.Object, "metadata", "labels", "app.edge-orchestrator.intel.com/deployment-id")
		if found {
			var ok bool
			deploymentName, ok = deploymentNameMap[deploymentID]
			if !ok {
				deploymentName = deploymentID
			}
		}

		conditionType := ""
		for _, condition := range conditions {
			condMap, ok := condition.(map[string]interface{})
			if ok {
				if reason, found := condMap["reason"]; found {
					if reason == "Error" {
						if conditionType, found = condMap["type"].(string); !found {
							conditionType = "unknown"
						}

						table.Append([]string{projectName, clusterName, deploymentName, appName, conditionType, reason.(string), bundle.GetNamespace(), name})
						// stop on the first error found
						break
					}
				}
			}
		}
	}
	table.Render()
}

var namespaceResource = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}

func ListNamespaces(ctx context.Context, client dynamic.Interface) ([]string, error) {
	list, err := client.Resource(namespaceResource).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	namespaces := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		namespaces = append(namespaces, item.GetName())
	}

	return namespaces, nil
}

var bundleResource = schema.GroupVersionResource{Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "bundledeployments"}

func ListBundles(ctx context.Context, client dynamic.Interface, namespace string) ([]unstructured.Unstructured, error) {
	var list *unstructured.UnstructuredList
	var err error
	if namespace == "" {
		list, err = client.Resource(bundleResource).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = client.Resource(bundleResource).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

var projectResource = schema.GroupVersionResource{Group: "project.edge-orchestrator.intel.com", Version: "v1", Resource: "projects"}

func GetProjectNames(ctx context.Context, client dynamic.Interface) (map[string]string, error) {
	list, err := client.Resource(projectResource).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	projectMap := make(map[string]string, len(list.Items))
	for _, item := range list.Items {
		uid, found, _ := unstructured.NestedString(item.Object, "status", "projectStatus", "uID")
		if !found {
			fmt.Fprintf(os.Stderr, "Project item without UID: %+v\n", item.Object)
			continue
		}
		name, found, _ := unstructured.NestedString(item.Object, "metadata", "labels", "nexus/display_name")
		if !found {
			fmt.Fprintf(os.Stderr, "Project item without display name: %+v\n", item.Object)
			continue
		}
		projectMap[uid] = name
	}
	return projectMap, nil
}

var deploymentResource = schema.GroupVersionResource{Group: "app.edge-orchestrator.intel.com", Version: "v1beta1", Resource: "deployments"}

func GetDeploymentNames(ctx context.Context, client dynamic.Interface) (map[string]string, error) {
	list, err := client.Resource(deploymentResource).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	deploymentMap := make(map[string]string, len(list.Items))
	for _, item := range list.Items {
		name := string(item.GetUID())

		displayName, found, _ := unstructured.NestedString(item.Object, "spec", "displayName")
		if !found {
			fmt.Fprintf(os.Stderr, "Deployment item without display name: %+v\n", item.Object)
			continue
		}

		deploymentMap[name] = displayName
	}
	return deploymentMap, nil
}
