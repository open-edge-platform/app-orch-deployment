// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package basic is a suite of basic functionality tests for the ADM service
package vm

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/shared"
	"testing"
	"time"

	armClient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
	"github.com/stretchr/testify/suite"
)

const (
	VMRunning = string(armClient.VirtualMachineStatusStateSTATERUNNING)
	VMStopped = string(armClient.VirtualMachineStatusStateSTATESTOPPED)
)

// TestSuite is the basic test suite
type TestSuite struct {
	shared.BaseSuite
}

// SetupTest can be used for per-test setup if needed
func (s *TestSuite) SetupTest() {
	// Leave empty or add per-test setup logic here
}

// TearDownSuite cleans up after the entire test suite
func (s *TestSuite) TearDownSuite() {
	err := utils.DeleteAndRetryUntilDeleted(s.AdmClient, utils.CirrosAppName, 10, 10*time.Second)
	s.NoError(err)
	err = utils.DeleteAndRetryUntilDeleted(s.AdmClient, utils.VirtualizationExtensionAppName, 10, 10*time.Second)
	s.NoError(err)
	utils.TearDownPortForward(s.PortForwardCmd)

}

func TestVMSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
