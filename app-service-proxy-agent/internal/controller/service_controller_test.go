// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Cluster Controller", func() {
	const (
		testServiceCR1Name       = "test1"
		testServiceCR1Namespace  = "default"
		testServiceCR1ClusterIP  = "10.0.0.100"
		testServiceCR1PortName   = "http-test1"
		testServiceCR1PortNumber = 8080

		testServiceCR2Name       = "test2"
		testServiceCR2Namespace  = "default"
		testServiceCR2ClusterIP  = "10.0.0.101"
		testServiceCR2PortName   = "http-test2"
		testServiceCR2PortNumber = 8081

		testServiceCR3Name       = "test3"
		testServiceCR3Namespace  = "default"
		testServiceCR3ClusterIP  = "10.0.0.102"
		testServiceCR3PortName   = "http-test3"
		testServiceCR3PortNumber = 8082

		testServiceCR4Name       = "test4"
		testServiceCR4Namespace  = "default"
		testServiceCR4ClusterIP  = "10.0.0.103"
		testServiceCR4PortName   = "http-test4"
		testServiceCR4PortNumber = 8083

		interval = 1 * time.Second
		timeout  = 10 * interval
	)

	var (
		testServiceCR1 *v1.Service
		testServiceCR2 *v1.Service
		testServiceCR3 *v1.Service
		testServiceCR4 *v1.Service
	)

	BeforeEach(func() {
		testServiceCR1 = &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testServiceCR1Name,
				Namespace: testServiceCR1Namespace,
				Annotations: map[string]string{
					"service-proxy.app.orchestrator.io/ports": fmt.Sprintf("%d", testServiceCR1PortNumber),
				},
			},
			Spec: v1.ServiceSpec{
				ClusterIP: testServiceCR1ClusterIP,
				Ports: []v1.ServicePort{
					{
						Name: testServiceCR1PortName,
						Port: testServiceCR1PortNumber,
					},
				},
			},
		}

		testServiceCR2 = &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testServiceCR2Name,
				Namespace: testServiceCR2Namespace,
			},
			Spec: v1.ServiceSpec{
				ClusterIP: testServiceCR2ClusterIP,
				Ports: []v1.ServicePort{
					{
						Name: testServiceCR2PortName,
						Port: testServiceCR2PortNumber,
					},
				},
			},
		}

		testServiceCR3 = &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testServiceCR3Name,
				Namespace: testServiceCR3Namespace,
				Annotations: map[string]string{
					"service-proxy.app.orchestrator.io/ports": "test1",
				},
			},
			Spec: v1.ServiceSpec{
				ClusterIP: testServiceCR3ClusterIP,
				Ports: []v1.ServicePort{
					{
						Name: testServiceCR3PortName,
						Port: testServiceCR3PortNumber,
					},
				},
			},
		}

		testServiceCR4 = &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testServiceCR4Name,
				Namespace: testServiceCR4Namespace,
				Annotations: map[string]string{
					"service-proxy.app.orchestrator.io/ports": fmt.Sprintf("%d,%d", testServiceCR1PortNumber, testServiceCR4PortNumber),
				},
			},
			Spec: v1.ServiceSpec{
				ClusterIP: testServiceCR4ClusterIP,
				Ports: []v1.ServicePort{
					{
						Name: testServiceCR4PortName,
						Port: testServiceCR4PortNumber,
					},
				},
			},
		}
	})

	AfterEach(func() {
	})

	It("a new service created with annotation", func() {
		Expect(k8sClient.Create(ctx, testServiceCR1)).Should(Succeed())

		Eventually(func() bool {
			keyService := types.NamespacedName{
				Namespace: testServiceCR1Namespace,
				Name:      testServiceCR1Name,
			}
			keyRole := types.NamespacedName{
				Namespace: testServiceCR1Namespace,
				Name:      getRoleName(testServiceCR1),
			}
			keyRoleBinding := types.NamespacedName{
				Namespace: testServiceCR1Namespace,
				Name:      getRoleName(testServiceCR1),
			}
			service := &v1.Service{}
			err := k8sClient.Get(ctx, keyService, service)
			if err != nil {
				return err == nil
			}
			role := &rbacv1.Role{}
			err = k8sClient.Get(ctx, keyRole, role)
			if err != nil {
				return err == nil
			}
			roleBinding := &rbacv1.RoleBinding{}
			err = k8sClient.Get(ctx, keyRoleBinding, roleBinding)
			if err != nil {
				return err == nil
			}

			return true
		}, timeout, interval).Should(BeTrue())
	})

	It("a new service created without annotation", func() {
		Expect(k8sClient.Create(ctx, testServiceCR2)).Should(Succeed())

		Eventually(func() bool {
			keyService := types.NamespacedName{
				Namespace: testServiceCR2Namespace,
				Name:      testServiceCR2Name,
			}
			keyRole := types.NamespacedName{
				Namespace: testServiceCR2Namespace,
				Name:      getRoleName(testServiceCR2),
			}
			keyRoleBinding := types.NamespacedName{
				Namespace: testServiceCR2Namespace,
				Name:      getRoleName(testServiceCR2),
			}
			service := &v1.Service{}
			err := k8sClient.Get(ctx, keyService, service)
			if err != nil {
				return err == nil
			}
			role := &rbacv1.Role{}
			err = k8sClient.Get(ctx, keyRole, role)
			if err != nil {
				return err == nil
			}
			roleBinding := &rbacv1.RoleBinding{}
			err = k8sClient.Get(ctx, keyRoleBinding, roleBinding)
			return err == nil
		}, timeout, interval).Should(BeFalse())
	})

	It("a new service created with wrong port in annotation", func() {
		Expect(k8sClient.Create(ctx, testServiceCR3)).Should(Succeed())

		Eventually(func() bool {
			keyService := types.NamespacedName{
				Namespace: testServiceCR3Namespace,
				Name:      testServiceCR3Name,
			}
			keyRole := types.NamespacedName{
				Namespace: testServiceCR3Namespace,
				Name:      getRoleName(testServiceCR3),
			}
			keyRoleBinding := types.NamespacedName{
				Namespace: testServiceCR3Namespace,
				Name:      getRoleName(testServiceCR3),
			}
			service := &v1.Service{}
			err := k8sClient.Get(ctx, keyService, service)
			if err != nil {
				return false
			}
			role := &rbacv1.Role{}
			err = k8sClient.Get(ctx, keyRole, role)
			if err != nil {
				return false
			}
			roleBinding := &rbacv1.RoleBinding{}
			err = k8sClient.Get(ctx, keyRoleBinding, roleBinding)
			return err == nil
		}, timeout, interval).Should(BeFalse())
	})

	It("a new service created with mismatched port between annotation and service spec", func() {
		Expect(k8sClient.Create(ctx, testServiceCR4)).Should(Succeed())

		Eventually(func() bool {
			keyService := types.NamespacedName{
				Namespace: testServiceCR4Namespace,
				Name:      testServiceCR4Name,
			}
			keyRole := types.NamespacedName{
				Namespace: testServiceCR4Namespace,
				Name:      getRoleName(testServiceCR4),
			}
			keyRoleBinding := types.NamespacedName{
				Namespace: testServiceCR4Namespace,
				Name:      getRoleName(testServiceCR4),
			}
			service := &v1.Service{}
			err := k8sClient.Get(ctx, keyService, service)
			if err != nil {
				return false
			}
			role := &rbacv1.Role{}
			err = k8sClient.Get(ctx, keyRole, role)
			if err != nil {
				return false
			}
			roleBinding := &rbacv1.RoleBinding{}
			err = k8sClient.Get(ctx, keyRoleBinding, roleBinding)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	})
})
