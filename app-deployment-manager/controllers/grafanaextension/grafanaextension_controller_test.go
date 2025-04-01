// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package grafanaextension

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("GrafanaExtension controller", Ordered, func() {

	const (
		testGrafanaNamespace = "default"

		testGrafanaCR1Name                    = "grafana1"
		testGrafanaCR1DisplayName             = "grafana1_display"
		testGrafanaCR1Project                 = "grafana1_project"
		testGrafanaCR1ArtifactPublisher       = "publisher1"
		testGrafanaCR1ArtifactName            = "artifact1"
		testGrafanaCR1ArtifactDescription     = "artifact description 1"
		testGrafanaCR1ArtifactContents        = "{\"uid\": \"foo1\"}"
		testGrafanaCR1ArtifactContentsUpdated = "{\"uid\": \"bar1\"}"

		testGrafanaCR2Name                = "grafana2"
		testGrafanaCR2DisplayName         = "grafana2_display"
		testGrafanaCR2Project             = "grafana2_project"
		testGrafanaCR2ArtifactPublisher   = "publisher2"
		testGrafanaCR2ArtifactName        = "artifact2"
		testGrafanaCR2ArtifactDescription = "artifact description 2"
		testGrafanaCR2ArtifactContents    = "{\"uid\": \"foo2\"}"

		testGrafanaCR3Name                = "grafana3"
		testGrafanaCR3DisplayName         = "grafana3_display"
		testGrafanaCR3Project             = "grafana3_project"
		testGrafanaCR3ArtifactPublisher   = "publisher3"
		testGrafanaCR3ArtifactName        = "artifact3"
		testGrafanaCR3ArtifactDescription = "artifact description 3"
		testGrafanaCR3ArtifactContents    = "{\"uid\": \"foo3\"}"

		interval = retryInterval * 2 * retryUpdateCounter
		timeout  = interval * 10
	)

	var (
		testGrafanaCR1 *v1beta1.GrafanaExtension
		testGrafanaCR2 *v1beta1.GrafanaExtension
		testGrafanaCR3 *v1beta1.GrafanaExtension
		//funcVarMutex   = sync.RWMutex{}
	)

	BeforeEach(OncePerOrdered, func() {
		testGrafanaCR1 = &v1beta1.GrafanaExtension{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testGrafanaCR1Name,
				Namespace: testGrafanaNamespace,
			},
			Spec: v1beta1.GrafanaExtensionSpec{
				DisplayName: testGrafanaCR1DisplayName,
				Project:     testGrafanaCR1Project,
				ArtifactRef: v1beta1.ArtifactRef{
					Publisher:   testGrafanaCR1ArtifactPublisher,
					Name:        testGrafanaCR1ArtifactName,
					Description: testGrafanaCR1ArtifactDescription,
					Artifact:    testGrafanaCR1ArtifactContents,
				},
			},
			Status: v1beta1.GrafanaExtensionStatus{},
		}

		testGrafanaCR2 = &v1beta1.GrafanaExtension{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testGrafanaCR2Name,
				Namespace: testGrafanaNamespace,
			},
			Spec: v1beta1.GrafanaExtensionSpec{
				DisplayName: testGrafanaCR2DisplayName,
				Project:     testGrafanaCR2Project,
				ArtifactRef: v1beta1.ArtifactRef{
					Publisher:   testGrafanaCR2ArtifactPublisher,
					Name:        testGrafanaCR2ArtifactName,
					Description: testGrafanaCR2ArtifactDescription,
					Artifact:    testGrafanaCR2ArtifactContents,
				},
			},
			Status: v1beta1.GrafanaExtensionStatus{},
		}

		testGrafanaCR3 = &v1beta1.GrafanaExtension{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testGrafanaCR3Name,
				Namespace: testGrafanaNamespace,
			},
			Spec: v1beta1.GrafanaExtensionSpec{
				DisplayName: testGrafanaCR3DisplayName,
				Project:     testGrafanaCR3Project,
				ArtifactRef: v1beta1.ArtifactRef{
					Publisher:   testGrafanaCR3ArtifactPublisher,
					Name:        testGrafanaCR3ArtifactName,
					Description: testGrafanaCR3ArtifactDescription,
					Artifact:    testGrafanaCR3ArtifactContents,
				},
			},
			Status: v1beta1.GrafanaExtensionStatus{},
		}
	})

	AfterEach(OncePerOrdered, func() {
	})

	It("a new GrafanaExtension CR is created and if the spec is valid - should create status", func() {
		funcGetGrafanaDashboardUIDMutex.Lock()
		funcGetGrafanaDashboardUID = func(ctx context.Context, dashboardJSONString string) (string, error) {
			return "", nil
		}
		funcGetGrafanaDashboardUIDMutex.Unlock()
		funcIsGrafanaDashboardReadyMutex.Lock()
		funcIsGrafanaDashboardReady = func(ctx context.Context, uid string) error {
			return nil
		}
		funcIsGrafanaDashboardReadyMutex.Unlock()

		// Create GrafanaExtension CR
		Expect(k8sClient.Create(ctx, testGrafanaCR1)).Should(Succeed())

		Eventually(func() bool {
			key := types.NamespacedName{
				Namespace: testGrafanaNamespace,
				Name:      testGrafanaCR1Name,
			}
			gext := &v1beta1.GrafanaExtension{}
			err := k8sClient.Get(ctx, key, gext)
			if err != nil {
				return false
			}
			return gext.Status.State == v1beta1.Running
		}, timeout, interval).Should(BeTrue())
	})
	It("a new GrafanaExtension CR is created and if the CR is already created and changed artifact contents - could update configmap", func() {
		funcGetGrafanaDashboardUIDMutex.Lock()
		funcGetGrafanaDashboardUID = func(ctx context.Context, dashboardJSONString string) (string, error) {
			return "", nil
		}
		funcGetGrafanaDashboardUIDMutex.Unlock()
		funcIsGrafanaDashboardReadyMutex.Lock()
		funcIsGrafanaDashboardReady = func(ctx context.Context, uid string) error {
			return nil
		}
		funcIsGrafanaDashboardReadyMutex.Unlock()

		// Get GrafanaExtension CR 1
		key := types.NamespacedName{
			Namespace: testGrafanaNamespace,
			Name:      testGrafanaCR1Name,
		}
		grafanaExtensionCR1Old := &v1beta1.GrafanaExtension{}
		Expect(k8sClient.Get(ctx, key, grafanaExtensionCR1Old)).Should(Succeed())

		// Update GrafanaExtension CR
		grafanaExtensionCR1Old.Spec.ArtifactRef.Artifact = testGrafanaCR1ArtifactContentsUpdated
		Expect(k8sClient.Update(ctx, grafanaExtensionCR1Old)).Should(Succeed())

		Eventually(func() bool {
			gextKey := types.NamespacedName{
				Namespace: testGrafanaNamespace,
				Name:      testGrafanaCR1Name,
			}
			gext := &v1beta1.GrafanaExtension{}
			err := k8sClient.Get(ctx, gextKey, gext)
			if err != nil {
				return false
			}

			jsonFileName := fmt.Sprintf("%s-%s.json", gext.UID, gext.Name)
			cmKey := types.NamespacedName{
				Namespace: gext.Namespace,
				Name:      fmt.Sprintf("%s-%s", gext.Name, gext.UID),
			}
			cm := &v1.ConfigMap{}
			err = k8sClient.Get(ctx, cmKey, cm)
			if err != nil {
				return false
			}

			return cm.Data[jsonFileName] == testGrafanaCR1ArtifactContentsUpdated
		}, timeout, interval).Should(BeTrue())
	})

	It("a new GrafanaExtension CR is created and if the spec is already created - could not find GrafanaExtension CR", func() {
		funcGetGrafanaDashboardUIDMutex.Lock()
		funcGetGrafanaDashboardUID = func(ctx context.Context, dashboardJSONString string) (string, error) {
			return "", nil
		}
		funcGetGrafanaDashboardUIDMutex.Unlock()
		funcIsGrafanaDashboardReadyMutex.Lock()
		funcIsGrafanaDashboardReady = func(ctx context.Context, uid string) error {
			return nil
		}
		funcIsGrafanaDashboardReadyMutex.Unlock()

		// Create GrafanaExtension CR
		Expect(k8sClient.Delete(ctx, testGrafanaCR1)).Should(Succeed())

		Eventually(func() bool {
			key := types.NamespacedName{
				Namespace: testGrafanaNamespace,
				Name:      testGrafanaCR1Name,
			}
			gext := &v1beta1.GrafanaExtension{}
			err := k8sClient.Get(ctx, key, gext)
			return err != nil
		}, timeout, interval).Should(BeTrue())
	})

	It("a Grafana dashboard status check is failed and if Grafana reconcile is failed to get Grafana Dashboard UID - should be failed", func() {
		funcGetGrafanaDashboardUIDMutex.Lock()
		funcGetGrafanaDashboardUID = func(ctx context.Context, dashboardJSONString string) (string, error) {
			return "", fmt.Errorf("err")
		}
		funcGetGrafanaDashboardUIDMutex.Unlock()
		funcIsGrafanaDashboardReadyMutex.Lock()
		funcIsGrafanaDashboardReady = func(ctx context.Context, uid string) error {
			return nil
		}
		funcIsGrafanaDashboardReadyMutex.Unlock()

		Expect(k8sClient.Create(ctx, testGrafanaCR2)).Should(Succeed())

		Eventually(func() bool {
			key := types.NamespacedName{
				Namespace: testGrafanaNamespace,
				Name:      testGrafanaCR2Name,
			}
			gext := &v1beta1.GrafanaExtension{}
			err := k8sClient.Get(ctx, key, gext)
			if err != nil {
				return false
			}

			return gext.Status.State == v1beta1.Down && gext.Status.Display == "Down:DashboardReady(False)"
		}, timeout, interval).Should(BeTrue())
	})

	It("a Grafana dashboard status check is failed and if Grafana reconcile is failed to get Grafana status - should be failed", func() {
		funcGetGrafanaDashboardUIDMutex.Lock()
		funcGetGrafanaDashboardUID = func(ctx context.Context, dashboardJSONString string) (string, error) {
			return "", nil
		}
		funcGetGrafanaDashboardUIDMutex.Unlock()
		funcIsGrafanaDashboardReadyMutex.Lock()
		funcIsGrafanaDashboardReady = func(ctx context.Context, uid string) error {
			return fmt.Errorf("err")
		}
		funcIsGrafanaDashboardReadyMutex.Unlock()

		Expect(k8sClient.Create(ctx, testGrafanaCR3)).Should(Succeed())

		Eventually(func() bool {
			key := types.NamespacedName{
				Namespace: testGrafanaNamespace,
				Name:      testGrafanaCR3Name,
			}
			gext := &v1beta1.GrafanaExtension{}
			err := k8sClient.Get(ctx, key, gext)
			if err != nil {
				return false
			}

			return gext.Status.State == v1beta1.Down && gext.Status.Display == "Down:DashboardReady(False)"
		}, timeout, interval).Should(BeTrue())
	})
})
