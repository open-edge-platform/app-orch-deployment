// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package servicelink

import (
	"context"
	"encoding/json"
	"fmt"
	//"os"
	"strings"
	"time"
	//"path/filepath"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
)

func GetCliSecretHarbor(url, token string) (string, error) {
	// Create a user data dir for Chrome to persist cookies and session
	//userDataDir := filepath.Join(os.TempDir(), "chromedp-user-data")
	//os.MkdirAll(userDataDir, 0700)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("no-sandbox", true),
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

	err = chromedp.Cancel(ctx)
	if err != nil {
		fmt.Println("chromedp cancel failed")
	}
	return secret, nil
}

func OpenPageInHeadlessChrome(url, search, _ string) (bool, error) {
	// Create a user data dir for Chrome to persist cookies and session
	//userDataDir := filepath.Join(os.TempDir(), "chromedp-user-data")
	//os.MkdirAll(userDataDir, 0700)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("no-sandbox", true),
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

	/*selectorPassword := "#password" // change as needed
	    selectorLogin := "#login" // change as needed
	    selectors := []string{"#username", "#password", "#kc-login",
	        "input-error-container-username", "input-error-container-password",
	        "id-hidden-input", "kc-form-login", "kc-login"}

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
			return false, fmt.Errorf("js error: %w", err)
	    }

	    // Parse and print the result
	    for _, res := range results {
	        fmt.Println(res)
	    }*/

	// Listen for console events
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				fmt.Println("Console:", arg.Value)
			}
		}
	})
	// Listen for network events
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSent:
			fmt.Printf("Request: %s %s\n", ev.Request.Method, ev.Request.URL)
			if cookie, found := ev.Request.Headers["Cookie"]; found {
				fmt.Printf("Request URL: %s\nCookie header: %s\n\n", ev.Request.URL, cookie)
			}
			if cookie, found := ev.Request.Headers["access-control-allow-origin"]; found {
				fmt.Printf("Request URL: %s\nCookie header: %s\n\n", ev.Request.URL, cookie)
			}
		case *network.EventResponseReceived:
			fmt.Printf("Response: %d %s\n", ev.Response.Status, ev.Response.URL)
			if cookie, found := ev.Response.Headers["Cookie"]; found {
				fmt.Printf("Response URL: %s\nCookie header: %s\n\n",
					ev.Response.URL, cookie)
			}
			if cookie, found := ev.Response.Headers["access-control-allow-origin"]; found {
				fmt.Printf("Response URL: %s\nCookie header: %s\n\n",
					ev.Response.URL, cookie)
			}
		}
	})

	// The selector for the <p> element
	selector := "p.has-text-align-center"

	var html string

	// Run tasks
	err := chromedp.Run(ctx,
		//network.Enable(),
		//network.SetExtraHTTPHeaders(network.Headers{
		//	"Authorization": "Bearer " + token,
		//}),
		chromedp.Navigate(url), // Navigate to the URL
		//chromedp.WaitVisible("body", chromedp.ByQuery), // Wait for the page body to be visible
		//chromedp.Evaluate(`
		//Array.from(document.querySelectorAll('[id]')).map(el => el.id)`, &ids),
		chromedp.WaitReady("body"),
		chromedp.SendKeys(`#username`, username, chromedp.ByID),
		chromedp.SendKeys(`#password`, password, chromedp.ByID),
		chromedp.Click(`#kc-login`, chromedp.ByID),
		chromedp.Sleep(3*time.Second),
		chromedp.CaptureScreenshot(&buf),
		//chromedp.WaitReady("body"),
		chromedp.Title(&pageTitle), // Get the page title
		chromedp.WaitVisible(selector),
		chromedp.InnerHTML(selector, &html, chromedp.NodeVisible),
	)

	if err != nil {
		return false, fmt.Errorf("chromedp error: %w", err)
	}

	found := false
	// Check for your string in the element's HTML
	if strings.Contains(html, search) {
		fmt.Println("String found!")
		found = true
	} else {
		fmt.Println("String not found.")
	}

	fmt.Println("html link : ", pageTitle)
	// Get all cookies visible to the current page
	var cookies []*network.Cookie
	err = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)
	if err != nil {
		return false, fmt.Errorf("chromedp error: %w", err)
	}

	// Print cookie information
	fmt.Println("Cookies (as seen in Application tab):")
	for _, c := range cookies {
		fmt.Printf("Name: %s\nValue: %s\nDomain: %s\nPath: %s\nExpires: %v\nSecure: %v\nHttpOnly: %v\n\n",
			c.Name, c.Value, c.Domain, c.Path, c.Expires, c.Secure, c.HTTPOnly)
	}

	err = chromedp.Cancel(ctx)
	if err != nil {
		fmt.Println("chromedp cancel failed")
	}
	return found, nil
}

func AppEndpointsList(armClient *restClient.ClientWithResponses, appID string) (*[]restClient.ResourceV2AppEndpoint, int, error) {
	resp, err :=
		armClient.ResourceV2EndpointsServiceListAppEndpointsWithResponse(context.TODO(),
			appID, types.TestClusterID)
	if err != nil || resp.StatusCode() != 200 {
		if err != nil {
			return &[]restClient.ResourceV2AppEndpoint{}, resp.StatusCode(), fmt.Errorf("%v", err)
		}
		return &[]restClient.ResourceV2AppEndpoint{}, resp.StatusCode(), fmt.Errorf("failed to list app endpoints: %v", string(resp.Body))
	}

	return resp.JSON200.AppEndpoints, resp.StatusCode(), nil
}
