// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// mockProjectResolverTest implements ProjectResolver for tests.
type mockProjectResolverTest struct {
	projects map[string]struct {
		orgName     string
		projectName string
	}
}

func (m *mockProjectResolverTest) ResolveProject(_ context.Context, projectID string) (string, string, error) {
	p, ok := m.projects[projectID]
	if !ok {
		return "", "", fmt.Errorf("project %s not found", projectID)
	}
	return p.orgName, p.projectName, nil
}

var (
	cfg             *rest.Config
	k8sClient       client.Client
	projectResolver ProjectResolver
	testEnv         *envtest.Environment
	ctx             context.Context
	cancel          context.CancelFunc
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

	projectResolver = &mockProjectResolverTest{
		projects: map[string]struct {
			orgName     string
			projectName string
		}{
			"64f42b12-af68-4676-a689-657dd670daab": {
				orgName:     "mock-org-1",
				projectName: "mock-project-1-in-mock-org-1",
			},
			"0a07df38-9df6-4d91-8275-d907c27915b8": {
				orgName:     "mock-org-2",
				projectName: "mock-project-1-in-mock-org-2",
			},
		},
	}

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
