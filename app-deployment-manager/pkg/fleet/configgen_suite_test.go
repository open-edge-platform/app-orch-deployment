// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"context"
	fleetclient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet/mocks"
	v1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimeproject.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/client/clientset/versioned/typed/runtimeproject.edge-orchestrator.intel.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	NexusRuntimesLabel       = "runtimes.runtime.edge-orchestrator.intel.com"
	NexusRuntimeFoldersLabel = "runtimefolders.runtimefolder.edge-orchestrator.intel.com"
	NexusRuntimeMtLabel      = "multitenancies.tenancy.edge-orchestrator.intel.com"
)

var (
	cfg         *rest.Config
	k8sClient   client.Client
	nexusClient nexus.RuntimeprojectEdgeV1Interface
	testEnv     *envtest.Environment
	ctx         context.Context
	cancel      context.CancelFunc
)

func TestConfigGen(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "ConfigGen Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("../..", "config", "crd", "bases"),
			filepath.Join("../..", "pkg", "fleet", "test", "crd"),
		},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	nexusClient = fleetclient.NewMockNexusClient([]v1.RuntimeProject{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mock-runtime-project-hash-12345",
				Labels: map[string]string{
					NexusRuntimeMtLabel:      "default",
					"nexus/display_name":     "mock-project-1-in-mock-org-1",
					"nexus/is_name_hashed":   "true",
					NexusRuntimeFoldersLabel: "default",
					NexusOrgLabel:            "mock-org-1",
					NexusRuntimesLabel:       "default",
				},
				UID: "64f42b12-af68-4676-a689-657dd670daab",
			},
			Spec: v1.RuntimeProjectSpec{},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mock-runtime-project-hash-54321",
				Labels: map[string]string{
					NexusRuntimeMtLabel:      "default",
					"nexus/display_name":     "mock-project-1-in-mock-org-2",
					"nexus/is_name_hashed":   "true",
					NexusRuntimeFoldersLabel: "default",
					NexusOrgLabel:            "mock-org-2",
					NexusRuntimesLabel:       "default",
				},
				UID: "0a07df38-9df6-4d91-8275-d907c27915b8",
			},
			Spec: v1.RuntimeProjectSpec{},
		},
	})

	go func() {
		defer GinkgoRecover()
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
