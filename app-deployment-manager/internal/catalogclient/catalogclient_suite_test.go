// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package catalogclient_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var mockCtrl *gomock.Controller

func TestCatalogclient(t *testing.T) {
	RegisterFailHandler(Fail)

	mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	RunSpecs(t, "Catalogclient Suite")
}
