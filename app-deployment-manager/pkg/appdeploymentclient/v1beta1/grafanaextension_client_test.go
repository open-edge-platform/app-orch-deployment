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

var _ = Describe("Test Grafana Extension client", func() {
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

		It("successfully list grafana extension object", func() {
			var opts metav1.ListOptions

			ge := adClient.GrafanaExtensions("test_namespace")
			Expect(ge).NotTo(BeNil())
			resp, err := ge.List(ctx, opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})

		It("successfully get grafana extension object", func() {
			var opts metav1.GetOptions

			ge := adClient.GrafanaExtensions("test_namespace")
			Expect(ge).NotTo(BeNil())
			resp, err := ge.Get(ctx, "test", opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel() // Cancel ctx as soon as handleSearch returns.
		})

		It("successfully create grafana extension object", func() {
			var opts metav1.CreateOptions

			ge := adClient.GrafanaExtensions("test_namespace")
			Expect(ge).NotTo(BeNil())

			resp, err := ge.Create(ctx, nil, opts)
			Expect(resp).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})

		It("successfully delete grafana extension object", func() {
			var opts metav1.DeleteOptions

			ge := adClient.GrafanaExtensions("test_namespace")
			Expect(ge).NotTo(BeNil())
			err := ge.Delete(ctx, "test", opts)

			Expect(err).ShouldNot(HaveOccurred())
			defer cancel()
		})
	})

})
