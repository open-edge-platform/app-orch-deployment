// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/shared"

	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"github.com/stretchr/testify/suite"
)

// TestSuite is the basic test suite
type TestSuite struct {
	suite.Suite
	shared.BaseSuite
}

// SetupTest can be used for per-test setup if needed
func (s *TestSuite) SetupTest() {
	// Leave empty or add per-test setup logic here
}

// TearDownSuite cleans up after the entire test suite
func (s *TestSuite) TearDownSuite() {
	err := utils.DeleteAndRetryUntilDeleted(s.AdmClient, utils.NginxAppName, 10, 10*time.Second)
	s.NoError(err)
	utils.TearDownPortForward(s.PortForwardCmd)
}
