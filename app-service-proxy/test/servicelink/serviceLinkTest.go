// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package servicelink

import (
	"context"
	"fmt"
	"net/http"

	"testing"

	admClient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/test/deploy"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/test/utils"
)

// TestList tests both app workload and service endpoints
func (s *TestSuite) TestList() {
	for _, app := range s.deployApps {
		appID := *app.Id
		appEndpoints, retCode, err := AppEndpointsList(s.ArmClient, appID)
		s.NoError(err)
		s.NotEmpty(appEndpoints)
		for _, endPoint := range appEndpoints {
			s.NotEmpty(endPoint.Ports)
			for _, port := range endPoint.Ports {
				serviceUrl := port.ServiceProxyUrl
				if serviceUrl != nil {
					resp, err := http.Get(serviceUrl)
					if err != nil {
						s.T().Errorf("Failed to send request to %s: %s", serviceUrl, err)
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
					}
				}

			}
		}
	}
}
