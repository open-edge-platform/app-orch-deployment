// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package servicelink

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
    //"encoding/json"
	"strings"
	"time"
    //"os"
    //"path/filepath"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
    "github.com/chromedp/cdproto/runtime"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/test/utils"
	"github.com/sclevine/agouti"
)

const (
	retryDelay = 10 * time.Second
	retryCount = 10
)

func openPageInHeadlessChrome(url, search, token string) (bool, error) {
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

	var pageTitle, pageHTML string
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
    var text string
    var cookie string
    // The selector for the <p> element
    // Run tasks
	err := chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			"Authorization": "Bearer " + token,
		}),
		chromedp.Navigate(url),                           // Navigate to the URL
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
		chromedp.Title(&pageTitle),                     // Get the page title
        //chromedp.Sleep(3*time.Second),
        //chromedp.Evaluate(`document.cookie`, &cookie),
        //chromedp.Location(&text),
        //chromedp.Sleep(5*time.Second),
		//chromedp.CaptureScreenshot(&buf),
    )
	
    if err != nil {
		return false, fmt.Errorf("chromedp error: %w", err)
	}

    fmt.Println("Cookies on page:", cookie)
    fmt.Println("next url:", text)
    /*
    // Check for your string in the element's HTML
    searchString := "Études is a pioneering firm"
    if strings.Contains(text, searchString) {
        fmt.Println("String found!")
    } else {
        fmt.Println("String not found.")
    } 

    // Print the IDs
    for _, id := range ids {
        fmt.Println(id)
    }
*/
    
    if err := ioutil.WriteFile("/home/seu/temp/screenshot.png", buf, 0644); err != nil {
		return false, fmt.Errorf("screenshot error: %w", err)
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
	found := strings.Contains(pageHTML, search)
	return found, nil
}

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
