// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAppDeploymentClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "v1beta1 Suite")
}
