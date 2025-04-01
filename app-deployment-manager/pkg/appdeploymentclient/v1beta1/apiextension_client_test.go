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

var _ = Describe("Test API Extension client", func() {
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

		It("successfully list api extension object", func() {
			var opts metav1.ListOptions

			ae := adClient.APIExtensions("test_namespace")
			Expect(ae).NotTo(BeNil())
			resp, err := ae.List(ctx, opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})

		It("successfully get api extension object", func() {
			var opts metav1.GetOptions

			ae := adClient.APIExtensions("test_namespace")
			Expect(ae).NotTo(BeNil())
			resp, err := ae.Get(ctx, "test", opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel() // Cancel ctx as soon as handleSearch returns.
		})

		It("successfully create api extension object", func() {
			var opts metav1.CreateOptions

			ae := adClient.APIExtensions("test_namespace")
			Expect(ae).NotTo(BeNil())

			resp, err := ae.Create(ctx, nil, opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})

		It("successfully delete api extension object", func() {
			var opts metav1.DeleteOptions

			ae := adClient.APIExtensions("test_namespace")
			Expect(ae).NotTo(BeNil())
			err := ae.Delete(ctx, "test", opts)

			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})
	})

})
