// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
package gitclient

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGitclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gitclient Suite")
}
