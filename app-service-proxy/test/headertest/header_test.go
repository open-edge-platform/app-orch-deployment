// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package headertest

import (
	"strings"
)

func (s *TestSuite) TestAuthorizationHeader() {
	for _, app := range s.DeployApps {
		appID := *app.Id
		appName := *app.Name
		s.T().Logf("appName : %s", appName)
		if appName == "httpbin" {
			appEndPoints, retCode, err := AppEndpointsList(s.ArmClient, appID)
			s.Equal(retCode, 200)
			s.NoError(err)
			s.NotEmpty(appEndPoints)
			for _, appEndPoint := range *appEndPoints {
				if appEndPoint.Ports != nil {
					for _, port := range *appEndPoint.Ports {
						serviceUrl := port.ServiceProxyUrl
						if serviceUrl != nil && *serviceUrl != "" {
							preText, err := GetXAuthHeader(*serviceUrl, s.Token)
							if err != nil {
								s.T().Errorf("Failed to open httbin page : %s", err)
							}

							if strings.Contains(preText, "X-Auth-Copied") {
								s.T().Logf("Success: X-Auth-Copied found in <body><pre>!")
							} else {
								s.T().Errorf("Failure: X-Auth-Copied NOT found in <body><pre>.")
							}
						}
					}
				} else {
					s.T().Errorf("No ports in service")
				}
			}
		}
	}

}
