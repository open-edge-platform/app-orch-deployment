// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package headertest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
	//"path/filepath"
	"net/http"
	"strings"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
)

const (
	retryDelay = 10 * time.Second
	retryCount = 10
)

func getXAuthHeader(url, token string) (string, error) {
	// Create a user data dir for Chrome to persist cookies and session
	//userDataDir := filepath.Join(os.TempDir(), "chromedp-user-data")
	//os.MkdirAll(userDataDir, 0700)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("ignore-certificate-errors", true),
		//  chromedp.UserDataDir(userDataDir), // <-- this makes cookies persistent!
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var pageTitle string
	var buf []byte

	// CSS selector for the `/headers` link
	h2XPath := `//h2[@id="ENDPOINTS"]`
	liXPath := `//li[a[contains(@href, "//app-service-proxy.kind.internal/headers")]]`
	//preXPath := `//pre`

	//preSelector := `body > pre`
	var preText string

	username := "sample-project-edge-mgr"
	password := "ChangeMeOn1stLogin!"
	var cookies []*network.Cookie
	// Run tasks
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(url), // Navigate to the URL
		//chromedp.WaitReady("body"),
		chromedp.SendKeys(`#username`, username, chromedp.ByID),
		chromedp.SendKeys(`#password`, password, chromedp.ByID),
		chromedp.Click(`#kc-login`, chromedp.ByID),
		chromedp.CaptureScreenshot(&buf),
		chromedp.Sleep(3*time.Second),
		chromedp.CaptureScreenshot(&buf),
		chromedp.Title(&pageTitle), // Get the page title
		chromedp.WaitVisible(h2XPath),

		chromedp.ScrollIntoView(liXPath),
		chromedp.CaptureScreenshot(&buf),

		// Click on the URL
		chromedp.Click(fmt.Sprintf(`%s/a`, liXPath)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
		chromedp.CaptureScreenshot(&buf),
	)

	if err != nil {
		return "", fmt.Errorf("chromedp error: %w", err)
	}

	// Construct the HTTP request using net/http
	req, err := http.NewRequest("GET", "https://app-service-proxy.kind.internal/headers", nil) // Replace with your API endpoint
	if err != nil {
		return "", fmt.Errorf("Error creating http request: %w", err)
	}

	// Add cookies to the request header
	var cookieStrings []string
	for _, cookie := range cookies {
		cookieStrings = append(cookieStrings, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}
	if len(cookieStrings) > 0 {
		req.Header.Set("Cookie", strings.Join(cookieStrings, "; "))
	}

	// Create an HTTP client and execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error calling http client: %w", err)
	}
	defer resp.Body.Close() // Make sure to close the response body

	// Read the response body into a string
	bodyBytes, err := io.ReadAll(resp.Body) // Use io.ReadAll
	if err != nil {
		return "", fmt.Errorf("Error reading response nody: %w", err)
	}
	preText = string(bodyBytes)

	//preText = responseBody
	fmt.Println("html link : ", pageTitle)
	fmt.Println("preText : ", preText)
	if err := os.WriteFile("/tmp/httpbin.png", buf, 0644); err != nil {
		return "", fmt.Errorf("screenshot error: %w", err)
	}

	return preText, nil
}

func getCliSecretHarbor(url, token string) (string, error) {
	// Create a user data dir for Chrome to persist cookies and session
	//userDataDir := filepath.Join(os.TempDir(), "chromedp-user-data")
	//os.MkdirAll(userDataDir, 0700)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("ignore-certificate-errors", true),
		//  chromedp.UserDataDir(userDataDir), // <-- this makes cookies persistent!
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var pageTitle string
	var buf []byte
	// This will hold the IDs

	username := "sample-project-edge-mgr"
	password := "ChangeMeOn1stLogin!"

	selectors := []string{"#log_oidc"}

	js := `
    (function(selectors) {
        return selectors.map(function(selector) {
            var el = document.querySelector(selector);
            if (!el) return { selector: selector, exists: false };
            var tag = el.tagName.toLowerCase();
            var writable =
                (tag === 'input' || tag === 'textarea') &&
                !el.disabled &&
                !el.readOnly;
            var style = window.getComputedStyle(el);
            var hidden =
                style.display === 'none' ||
                style.visibility === 'hidden' ||
                el.offsetParent === null;
            var focusable = !el.disabled && !hidden &&
                (
                    ['input', 'select', 'textarea', 'button', 'a'].includes(tag) ||
                    el.hasAttribute('tabindex')
                );
            return {
                selector: selector,
                exists: true,
                writable: writable,
                hidden: hidden,
                focusable: focusable
            };
        });
    })(%s)
    `
	// Encode Go array as JSON for injection into JS
	selectorsJSON, _ := json.Marshal(selectors)
	js = fmt.Sprintf(js, string(selectorsJSON))

	var results []map[string]interface{}
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.Evaluate(js, &results),
	)
	if err != nil {
		return "", fmt.Errorf("js error: %w", err)
	}

	// Parse and print the result
	for _, res := range results {
		fmt.Println(res)
	}

	var secret string

	// Run tasks
	err = chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			"Authorization": "Bearer " + token,
		}),
		chromedp.Navigate(url), // Navigate to the URL
		chromedp.Sleep(3*time.Second),
		chromedp.CaptureScreenshot(&buf),
		chromedp.Click(`//button[normalize-space()='LOGIN WITH Open Edge IAM']`, chromedp.BySearch),
		chromedp.CaptureScreenshot(&buf),
		chromedp.WaitReady("body"),
		chromedp.SendKeys(`#username`, username, chromedp.ByID),
		chromedp.SendKeys(`#password`, password, chromedp.ByID),
		chromedp.Click(`#kc-login`, chromedp.ByID),
		chromedp.Sleep(3*time.Second),
		chromedp.CaptureScreenshot(&buf),
		chromedp.WaitVisible(`//button[span[contains(text(), "sample-project-edge-mgr")]]`, chromedp.BySearch),
		// Click the button
		chromedp.Click(`//button[span[contains(text(), "sample-project-edge-mgr")]]`, chromedp.BySearch),
		chromedp.Sleep(500*time.Millisecond), // Wait for menu animation/render
		chromedp.CaptureScreenshot(&buf),
		chromedp.WaitVisible(`//a[contains(text(), "User Profile")]`, chromedp.BySearch),
		chromedp.Click(`//a[contains(text(), "User Profile")]`, chromedp.BySearch),
		chromedp.Sleep(500*time.Millisecond), // Wait for menu animation/render
		chromedp.CaptureScreenshot(&buf),
		// Wait for the modal and textarea to appear
		chromedp.WaitVisible(`textarea.inputTarget`, chromedp.ByQuery),
		// Get the value of the textarea
		chromedp.Value(`textarea.inputTarget`, &secret, chromedp.ByQuery),
		chromedp.CaptureScreenshot(&buf),
	)

	if err != nil {
		return "", fmt.Errorf("chromedp error: %w", err)
	}

	fmt.Println("html link : ", pageTitle)
	fmt.Println("secret : ", secret)
	if err := os.WriteFile("/tmp/registry.png", buf, 0644); err != nil {
		return "", fmt.Errorf("screenshot error: %w", err)
	}

	return secret, nil
}

func AppEndpointsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.AppEndpoint, int, error) {
	resp, err :=
		armClient.EndpointsServiceListAppEndpointsWithResponse(context.TODO(),
			appID, types.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.AppEndpoint{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.AppEndpoint{}, resp.StatusCode(), fmt.Errorf("failed to list app endpoints: %v", string(resp.Body))
	}

	return resp.JSON200.AppEndpoints, resp.StatusCode(), nil
}
