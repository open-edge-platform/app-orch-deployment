// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Deployment client", func() {
	Describe("Calling API Methods", func() {
		var (
			ctx        context.Context
			cancel     context.CancelFunc
			restConfig *rest.Config
			fakeClient *http.Client
			adClient   *AppDeploymentClient
			err        error
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			restConfig = new(rest.Config)
			fakeClient = fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
				header := http.Header{}
				arr := []string{"1.1"}
				header.Set("Content-Type", runtime.ContentTypeJSON)
				return &http.Response{StatusCode: 200, Header: header,
					Body: objBody(&metav1.APIVersions{Versions: arr})}, nil
			})
			adClient, err = NewForConfig(restConfig)

			adClient.RESTClient().(*rest.RESTClient).Client = fakeClient
			Expect(adClient).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("successfully list deployments object", func() {
			var opts metav1.ListOptions

			deployments := adClient.Deployments("test_namespace")
			Expect(deployments).NotTo(BeNil())
			resp, err := deployments.List(ctx, opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})

		It("successfully get deployments object", func() {
			var opts metav1.GetOptions

			deployments := adClient.Deployments("test_namespace")
			Expect(deployments).NotTo(BeNil())
			resp, err := deployments.Get(ctx, "test", opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel() // Cancel ctx as soon as handleSearch returns.
		})

		It("successfully create deployments object", func() {
			var opts metav1.CreateOptions

			deployments := adClient.Deployments("test_namespace")
			Expect(deployments).NotTo(BeNil())

			resp, err := deployments.Create(ctx, nil, opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})

		It("successfully update deployments object", func() {
			var (
				opts  metav1.UpdateOptions
				gopts metav1.GetOptions
			)

			deployments := adClient.Deployments("test_namespace")
			Expect(deployments).NotTo(BeNil())

			resp, err := deployments.Get(ctx, "test", gopts)
			Expect(err).ShouldNot(HaveOccurred())

			resp, err = deployments.Update(ctx, "test", resp, opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})

		It("successfully delete deployments object", func() {
			var opts metav1.DeleteOptions

			deployments := adClient.Deployments("test_namespace")
			Expect(deployments).NotTo(BeNil())
			err := deployments.Delete(ctx, "test", opts)

			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})
	})

})
