// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&ServiceReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager, "proxy-name", "default")
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func TestHasAnnotationKey_FailedCheckingAnnotationKey(t *testing.T) {
	hasKey := hasAnnotationKey(nil, "")
	assert.Equal(t, false, hasKey)
}

func TestHasAnnotationKey(t *testing.T) {
	cr := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "default",
			Annotations: map[string]string{
				"service-proxy.app.orchestrator.io/ports": "8080",
			},
		},
	}

	hasKey := hasAnnotationKey(cr, "service-proxy.app.orchestrator.io/ports")
	assert.Equal(t, true, hasKey)
}

func TestGetResourceName_NilService(t *testing.T) {
	cr := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "name",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	}
	_, err := getResourceNames(cr)
	assert.Error(t, err)
}

func TestGetResourceName(t *testing.T) {
	cr := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "default",
			Annotations: map[string]string{
				"service-proxy.app.orchestrator.io/ports": "8080",
			},
		},
	}
	strs, err := getResourceNames(cr)
	assert.NotNil(t, strs)
	assert.NoError(t, err)
}
