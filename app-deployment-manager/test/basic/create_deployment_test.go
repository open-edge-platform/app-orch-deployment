// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/deploy"
)

const (
	wordpressAppName              = "wordpress"
	wordpressTargetedDisplayName  = "wordpress-targeted"
	wordpressAutoScaleDisplayName = "wordpress-auto-scaling"
)

func (s *TestSuite) TestCreateTargetedDeployment() {
	s.T().Parallel()

	_, err := deploy.CreateDeployment(s.AdmClient, wordpressAppName, wordpressTargetedDisplayName, 10)
	s.NoError(err, "Failed to create 'wordpress' deployment")
}

func (s *TestSuite) TestCreateAutoScaleDeployment() {
	s.T().Parallel()

	_, err := deploy.CreateDeployment(s.AdmClient, wordpressAppName, wordpressAutoScaleDisplayName, 10)
	s.NoError(err, "Failed to create 'wordpress' deployment")
}
