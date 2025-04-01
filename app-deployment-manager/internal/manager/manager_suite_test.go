// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestManager(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Manager Suite")
}
