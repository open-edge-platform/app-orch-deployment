// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/container"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/vm"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestAllSuites(t *testing.T) {
	suite.Run(t, new(vm.TestSuite))
	suite.Run(t, new(container.TestSuite))
}
