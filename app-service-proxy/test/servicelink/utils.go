// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package servicelink

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/test/utils"
	"github.com/sclevine/agouti"
)

const (
	retryDelay = 10 * time.Second
	retryCount = 10
)

func openPageInChrome(serviceUrl string) error {
	// Start a new WebDriver instance
	driver := agouti.ChromeDriver()
	if err := driver.Start(); err != nil {
		fmt.Println("Failed to start driver:", err)
		return err
	}
	defer driver.Stop()

	// Create a new page
	page, err := driver.NewPage()
	if err != nil {
		fmt.Println("Failed to open page:", err)
		return err
	}

	// Navigate to the URL
	fmt.Println("Navigating to URL:", serviceUrl)
	if err := page.Navigate(serviceUrl); err != nil {
		fmt.Println("Failed to navigate:", err)
		return err
	}

	// Check if the page is loaded
	if _, err := page.Find("body").Visible(); err != nil {
		fmt.Println("Page body not visible:", err)
		return err
	}
	// Find the element containing the specific text
	selection := page.Find("body").FindByXPath("//*[contains(text(), 'A commitment to innovation and sustainability')]")
	if visible, err := selection.Visible(); err != nil || !visible {
		fmt.Println("Text not found or not visible:", err)
		return err
	}

	// Retrieve and print the text
	text, err := selection.Text()
	if err != nil {
		fmt.Println("Failed to retrieve text:", err)
		return err
	}
	fmt.Println("Found text:", text)
	return nil

	/*
		// Interact with the page and execute JavaScript
		// For example, you can wait for an element to be visible
		if err := page.Find("#some-element").Visible(); err != nil {
			fmt.Println("Element not visible:", err)
			return
		}

		// You can also execute JavaScript directly
		result, err := page.RunScript("return document.title;", nil, nil)
		if err != nil {
			fmt.Println("Failed to run script:", err)
			return
		}
		fmt.Println("Page title is:", result) */
}
func AppEndpointsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.AppEndpoint, int, error) {
	resp, err := armClient.EndpointsServiceListAppEndpointsWithResponse(context.TODO(), appID, deploy.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.AppEndpoint{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.AppEndpoint{}, resp.StatusCode(), fmt.Errorf("failed to list app endpoints: %v", string(resp.Body))
	}

	return resp.JSON200.AppEndpoints, resp.StatusCode(), nil
}

func MethodAppWorkloadsList(verb, restServerURL, appID, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/workloads/%s/%s", restServerURL, appID, deploy.TestClusterID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}

func MethodAppEndpointsList(verb, restServerURL, appID, token, projectID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/resource.orchestrator.apis/v2/endpoints/%s/%s", restServerURL, appID, deploy.TestClusterID)
	res, err := utils.CallMethod(url, verb, token, projectID)
	if err != nil {
		return nil, err
	}

	return res, err
}
