// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const KIND = "deployments"

const VALID_CLUSTERID = "cluster-0123456789"
const VALID_PROJECT_ID = "0000-1111-2222-3333-4444"
const VALID_UID = "123456-123456-123456"
const NAMESPACE = "clusters"
const APIVersion = "v1"

var _ = Describe("Gateway Migration Service", func() {

	Describe("Gateway Migration", func() {
		It("successfully call migrate for deployments with no resources deployed", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "deployments", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockClient(ts.URL)

			deploymentServer := newMigration(kc, "test")

			fleetGVR := schema.GroupVersionResource{
				Group:    "app.edge-orchestrator.intel.com",
				Version:  "v1beta1",
				Resource: "",
			}

			deploymentServer.fleetGVR = fleetGVR

			err := deploymentServer.migrate(context.TODO(), "deployments")

			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully call start with no resources deployed", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "deployments", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockClient(ts.URL)

			deploymentServer := newMigration(kc, "test")

			fleetGVR := schema.GroupVersionResource{
				Group:    "app.edge-orchestrator.intel.com",
				Version:  "v1beta1",
				Resource: "",
			}

			deploymentServer.fleetGVR = fleetGVR

			err := deploymentServer.start(context.TODO())

			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func mockClient(tsUrl string) *dynamic.DynamicClient {
	config := &rest.Config{
		Host: tsUrl,
	}

	fleetGVR := schema.GroupVersion{
		Group:   "app.edge-orchestrator.intel.com",
		Version: "v1beta1",
	}

	config.GroupVersion = &fleetGVR

	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.ContentType = "application/json"

	kc, err := dynamic.NewForConfig(config)
	Expect(err).ToNot(HaveOccurred())

	return kc
}
