// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Cluster Controller", func() {

	const (
		testClusterNamespace = "default"

		testClusterCR1Name                            = "cluster1"
		testClusterCR1DisplayName                     = "MyCluster"
		testClusterCR1KubeConfigSecretName            = "kubeconfig1"
		testClusterCR1KubeConfigSecretContents        = "kubeconfig_contents1"         // #nosec G101
		testClusterCR1KubeConfigSecretNameUpdated     = "kubeconfig1updated"           // #nosec G101
		testClusterCR1KubeConfigSecretContentsUpdated = "kubeconfig_contents1_updated" // #nosec G101
		testClusterCR1BundleSummaryValues             = 1
		testClusterCR1BundleResourceCounts            = 2
		testClusterCR1FleetAgentStatusNamespace       = "cluster1"

		interval = 10 * time.Second
		timeout  = 100 * interval
	)

	var (
		testClusterCR1                 *fleetv1alpha1.Cluster
		testClusterCR1Status           *fleetv1alpha1.ClusterStatus
		testKubeConfigSecretCR1        *v1.Secret
		testKubeConfigSecretCR1Updated *v1.Secret
	)

	BeforeEach(func() {
		testClusterCR1 = &fleetv1alpha1.Cluster{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testClusterCR1Name,
				Namespace: testClusterNamespace,
				Labels: map[string]string{
					string(v1beta1.ClusterName): testClusterCR1DisplayName,
				},
			},
			Spec: fleetv1alpha1.ClusterSpec{
				KubeConfigSecret: testClusterCR1KubeConfigSecretName,
			},
		}

		testClusterCR1Status = &fleetv1alpha1.ClusterStatus{
			Summary: fleetv1alpha1.BundleSummary{
				Ready:        testClusterCR1BundleSummaryValues,
				DesiredReady: testClusterCR1BundleSummaryValues,
			},
			ResourceCounts: fleetv1alpha1.ResourceCounts{
				Ready:        testClusterCR1BundleResourceCounts,
				DesiredReady: testClusterCR1BundleResourceCounts,
				WaitApplied:  testClusterCR1BundleResourceCounts,
				Modified:     testClusterCR1BundleResourceCounts,
				Orphaned:     testClusterCR1BundleResourceCounts,
				Missing:      testClusterCR1BundleResourceCounts,
				Unknown:      testClusterCR1BundleResourceCounts,
				NotReady:     testClusterCR1BundleResourceCounts,
			},
			Display: fleetv1alpha1.ClusterDisplay{
				State: "ready",
			},
			Agent: fleetv1alpha1.AgentStatus{
				LastSeen: metav1.Time{
					Time: time.Now(),
				},
				Namespace: testClusterCR1FleetAgentStatusNamespace,
			}}

		testKubeConfigSecretCR1 = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testClusterCR1KubeConfigSecretName,
				Namespace: testClusterNamespace,
			},
			Data: map[string][]byte{"value": []byte(testClusterCR1KubeConfigSecretContents)},
		}

		testKubeConfigSecretCR1Updated = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testClusterCR1KubeConfigSecretNameUpdated,
				Namespace: testClusterNamespace,
			},
			Data: map[string][]byte{"value": []byte(testClusterCR1KubeConfigSecretContentsUpdated)},
		}
	})

	AfterEach(func() {
	})

	When("a new FleetCluster CR is created / updated / deleted", func() {
		Context("and if the spec is valid", func() {
			It("should create cluster CR", func() {
				Expect(k8sClient.Create(ctx, testKubeConfigSecretCR1)).Should(Succeed())
				Expect(k8sClient.Create(ctx, testClusterCR1)).Should(Succeed())

				createdClusterCR1 := &fleetv1alpha1.Cluster{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: testClusterCR1Name, Namespace: testClusterNamespace}, createdClusterCR1)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				createdClusterCR1.Status = *testClusterCR1Status

				Expect(k8sClient.Status().Update(ctx, createdClusterCR1)).Should(Succeed())

				Eventually(func() bool {
					key := types.NamespacedName{
						Namespace: testClusterNamespace,
						Name:      testClusterCR1Name,
					}
					c := &v1beta1.Cluster{}
					err := k8sClient.Get(ctx, key, c)
					if err != nil {
						return false
					}
					return c.Status.State == v1beta1.Running
				}, timeout, interval).Should(BeTrue())
			})
		})

		Context("and if the fleet cluster spec is updated", func() {
			It("should update cluster CR", func() {
				Expect(k8sClient.Create(ctx, testKubeConfigSecretCR1Updated)).Should(Succeed())

				createdClusterCR1 := &fleetv1alpha1.Cluster{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: testClusterCR1Name, Namespace: testClusterNamespace}, createdClusterCR1)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				createdClusterCR1.Spec.KubeConfigSecret = testClusterCR1KubeConfigSecretNameUpdated

				Expect(k8sClient.Update(ctx, createdClusterCR1)).Should(Succeed())

				Eventually(func() bool {
					key := types.NamespacedName{
						Namespace: testClusterNamespace,
						Name:      testClusterCR1Name,
					}
					c := &v1beta1.Cluster{}
					err := k8sClient.Get(ctx, key, c)
					if err != nil {
						return false
					}

					return c.Status.State == v1beta1.Running && c.Spec.KubeConfigSecretName == testClusterCR1KubeConfigSecretNameUpdated
				}, timeout, interval).Should(BeTrue())
			})
		})

		Context("and if the fleet cluster spec is deleted", func() {
			It("should delete cluster CR", func() {
				fc := &fleetv1alpha1.Cluster{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testClusterCR1Name, Namespace: testClusterNamespace}, fc)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, fc)).Should(Succeed())
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testClusterCR1Name, Namespace: testClusterNamespace}, fc))

				Eventually(func() bool {
					fc := &fleetv1alpha1.Cluster{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testClusterCR1Name, Namespace: testClusterNamespace}, fc)).ShouldNot(Succeed())

					c := &v1beta1.Cluster{}
					err := k8sClient.Get(ctx, types.NamespacedName{Name: testClusterCR1Name, Namespace: testClusterNamespace}, c)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			})
		})
	})
})
