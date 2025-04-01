// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMiddleware(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Middleware Suite")
}
