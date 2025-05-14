// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package servicelink

import (
	"net/http"
)

// TestList tests both app workload and service endpoints
func (s *TestSuite) TestList() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appEndPoints, retCode, err := AppEndpointsList(s.ArmClient, appID)
		s.Equal(retCode, 200)
		s.NoError(err)
		s.NotEmpty(appEndPoints)
		for _, appEndPoint := range *appEndPoints {
            if appEndPoint.Ports != nil {
                for _, port := range *appEndPoint.Ports {
                    serviceUrl := port.ServiceProxyUrl
                    if serviceUrl != nil {
                        resp, err := http.Get(*serviceUrl)
                        if err != nil {
                            s.T().Errorf("Failed to send request to %s: %s", *serviceUrl, err)
                        }
                        defer resp.Body.Close()

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
