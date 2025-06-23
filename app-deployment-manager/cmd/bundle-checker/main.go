// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()

	_ = verbose

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

	fmt.Printf("About to get namespaces\n")

	allBundles, err := ListBundles(context.Background(), dynamicClientSet, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list bundles: %s\n", err)
		os.Exit(1)
	}

	for _, bundle := range allBundles {
		name, found, _ := unstructured.NestedString(bundle.Object, "metadata", "name")
		if !found {
			continue
		}

		conditions, found, _ := unstructured.NestedSlice(bundle.Object, "status", "conditions")
		if !found {
			continue
		}

		hasError := false
		for _, condition := range conditions {
			condMap, ok := condition.(map[string]interface{})
			if ok {
				if reason, found := condMap["reason"]; found {
					if reason == "Error" {
						hasError = true
					}
				}
			}
		}
		if hasError {
			fmt.Printf("Bundle Name: %s, Namespace: %s\n", name, bundle.GetNamespace())
		}
	}
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
