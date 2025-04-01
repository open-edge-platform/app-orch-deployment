// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"bytes"
	"encoding/json"
	"io"

	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func objBody(object interface{}) io.ReadCloser {
	output, err := json.MarshalIndent(object, "", "")
	if err != nil {
		panic(err)
	}
	return io.NopCloser(bytes.NewReader(output))
}

var _ = Describe("Test App Deployment client", func() {
	Describe("Calling deployments", func() {
		It("successfully return new deployments", func() {
			restConfig := new(rest.Config)
			adClient, err := NewForConfig(restConfig)
			Expect(adClient).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			deployments := adClient.Deployments("test_namespace")
			Expect(deployments).NotTo(BeNil())
		})

		It("successfully creates a NewForConfig deploymentclient", func() {
			restConfig := new(rest.Config)
			deploymentclient, err := NewForConfig(restConfig)
			Expect(deploymentclient).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("successfully creates a NewForConfigAndClient deploymentclient", func() {
			restConfig := new(rest.Config)
			deploymentclient, err := NewForConfigAndClient(restConfig)
			Expect(deploymentclient).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("successfully creates a new deploymentclient", func() {
			restInterface := new(rest.Interface)
			deploymentclient := New(*restInterface)
			Expect(deploymentclient).NotTo(BeNil())
		})
	})
})
