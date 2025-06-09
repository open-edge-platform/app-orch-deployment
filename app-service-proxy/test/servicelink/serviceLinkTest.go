// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package servicelink

import (
	"io/ioutil"
	"net/http"
)

// TestServiceLink checks if http get for service link works.
func (s *TestSuite) TestServiceLinkPageAccess() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appName := *app.Name
		if appName == "wordpress" {
			appEndPoints, retCode, err := AppEndpointsList(s.ArmClient, appID)
			s.Equal(retCode, 200)
			s.NoError(err)
			s.NotEmpty(appEndPoints)
			for _, appEndPoint := range *appEndPoints {
				if appEndPoint.Ports != nil {
					for _, port := range *appEndPoint.Ports {
						serviceUrl := port.ServiceProxyUrl
						if serviceUrl != nil && *serviceUrl != "" {
							searchStr := "Études is a pioneering firm"
							found, err := openPageInHeadlessChrome(*serviceUrl, searchStr, s.Token)
							if err != nil || found == false {
								s.T().Errorf("Failed to open wordpress page : %s", err)
							}
						}
					}
				} else {
					s.T().Logf("No ports in service")
				}
			}
		}
	}
}

// TestServiceLink checks if http get for service link works.
func (s *TestSuite) TestServiceLink() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appEndPoints, retCode, err := AppEndpointsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appEndPoints)
		for _, appEndPoint := range *appEndPoints {
			if appEndPoint.Ports != nil {
				for _, port := range *appEndPoint.Ports {
					serviceUrl := port.ServiceProxyUrl
					if serviceUrl != nil && *serviceUrl != "" {
						resp, err := http.Get(*serviceUrl)
						if err != nil {
							s.T().Errorf("Failed to send request to %s: %s", *serviceUrl, err)
						}
						defer resp.Body.Close()

						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							s.T().Errorf("response body read error : %s", err)
						} else {
							s.T().Logf("Status Code: %d, %s", resp.StatusCode, http.StatusText(resp.StatusCode))
							s.T().Logf("Response Body: %s", string(body))
						}

						if resp.StatusCode != http.StatusOK {
							s.T().Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
						}
					}

				}
			} else {
				s.T().Logf("No ports in service")
			}
		}
	}
}

func (s *TestSuite) TestKeycloakRedirect() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appEndPoints, retCode, err := AppEndpointsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appEndPoints)
		for _, appEndPoint := range *appEndPoints {
			if appEndPoint.Ports != nil {
				for _, port := range *appEndPoint.Ports {
					serviceUrl := port.ServiceProxyUrl
					if serviceUrl != nil && *serviceUrl != "" {
						resp, err :=
							http.Get("https://app-service-proxy.kind.internal/app-service-proxy-main.js")
						if err != nil {
							s.T().Errorf("Failed to send request to %s: %s", *serviceUrl, err)
						}
						defer resp.Body.Close()

						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							s.T().Errorf("response body read error : %s", err)
						} else {
							s.T().Logf("Status Code: %d, %s", resp.StatusCode, http.StatusText(resp.StatusCode))
							s.T().Logf("Response Body: %s", string(body))
						}

						if resp.StatusCode != http.StatusOK {
							s.T().Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
						}
					}

				}
			} else {
				s.T().Logf("No ports in service")
			}
		}
	}
}
