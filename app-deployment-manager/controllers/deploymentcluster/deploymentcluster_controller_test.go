// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deploymentcluster

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/genericcondition"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

var _ = Describe("DeploymentCluster controller", func() {

	const (
		namespace = "default"

		deploymentId       = "a563356a-b4df-47bb-b620-aae2d74c5129"
		activeProjectId    = "a563356a-b4df-47bb-b620-aae2d74c5129"
		clusterId          = "cluster-12345"
		depClusterName     = "dc-70d44698-f8e0-54d5-90d7-8c121ed85e15"
		clusterDisplayName = "My Cluster"

		appName1    = "app-1"
		bdName1     = "bundledeployment-1"
		bundleName1 = "bundle-1"

		appName2    = "app-2"
		appMessage2 = "I am not ready"
		bdName2     = "bundledeployment-2"
		bundleName2 = "bundle-2"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		bd1           *fleetv1alpha1.BundleDeployment
		bd2           *fleetv1alpha1.BundleDeployment
		cluster       *v1beta1.Cluster
		clusterstatus *v1beta1.ClusterStatus
	)

	BeforeEach(func() {
		var jsonmap map[string]any

		bd1 = &fleetv1alpha1.BundleDeployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      bdName1,
				Namespace: namespace,
				Labels: map[string]string{
					string(v1beta1.AppName):                appName1,
					string(v1beta1.BundleName):             bundleName1,
					string(v1beta1.BundleType):             fleet.BundleTypeApp.String(),
					string(v1beta1.DeploymentID):           deploymentId,
					string(v1beta1.FleetClusterID):         clusterId,
					string(v1beta1.FleetClusterNamespace):  namespace,
					string(v1beta1.AppOrchActiveProjectID): activeProjectId,
					string(v1beta1.DeploymentGeneration):   "1",
				},
			},
			Spec: fleetv1alpha1.BundleDeploymentSpec{
				Options: fleetv1alpha1.BundleDeploymentOptions{
					Helm: &fleetv1alpha1.HelmOptions{
						Values: &fleetv1alpha1.GenericMap{
							Data: jsonmap,
						},
					},
				},
			},
			Status: fleetv1alpha1.BundleDeploymentStatus{
				Conditions: []genericcondition.GenericCondition{
					{
						Type:   fleetv1alpha1.BundleDeploymentConditionReady,
						Status: v1.ConditionTrue,
					},
				},
				Ready:       true,
				NonModified: true,
			},
		}
		bd2 = &fleetv1alpha1.BundleDeployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      bdName2,
				Namespace: namespace,
				Labels: map[string]string{
					string(v1beta1.AppName):               appName2,
					string(v1beta1.BundleName):            bundleName2,
					string(v1beta1.BundleType):            fleet.BundleTypeApp.String(),
					string(v1beta1.DeploymentID):          deploymentId,
					string(v1beta1.FleetClusterID):        clusterId,
					string(v1beta1.FleetClusterNamespace): namespace,
					string(v1beta1.DeploymentGeneration):  "1",
				},
			},
			Spec: fleetv1alpha1.BundleDeploymentSpec{
				Options: fleetv1alpha1.BundleDeploymentOptions{
					Helm: &fleetv1alpha1.HelmOptions{
						Values: &fleetv1alpha1.GenericMap{
							Data: jsonmap,
						},
					},
				},
			},
			Status: fleetv1alpha1.BundleDeploymentStatus{
				Conditions: []genericcondition.GenericCondition{
					{
						Type:    fleetv1alpha1.BundleDeploymentConditionReady,
						Status:  v1.ConditionFalse,
						Message: appMessage2,
					},
				},
				Ready:       false,
				NonModified: true,
			},
		}
		cluster = &v1beta1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterId,
				Namespace: namespace,
			},
			Spec: v1beta1.ClusterSpec{
				DisplayName: "My Cluster",
			},
		}
		clusterstatus = &v1beta1.ClusterStatus{
			Display: clusterDisplayName,
			State:   v1beta1.Running,
			FleetStatus: v1beta1.FleetStatus{
				BundleSummary:  v1beta1.BundleSummary{},
				ResourceCounts: v1beta1.ResourceCounts{},
				FleetAgentStatus: v1beta1.FleetAgentStatus{
					LastSeen:  metav1.Now(),
					Namespace: namespace,
				},
				ClusterDisplay: v1beta1.ClusterDisplay{},
			},
		}
	})

	AfterEach(func() {
	})

	When("fetching the DeploymentCluster info for a BundleDeployment", func() {

		Context("the correct labels are present", func() {
			It("should return the expected info", func() {
				dci, err := deploymentClusterInfo(&fleetv1alpha1.BundleDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "good-bundledeployment",
						Namespace: "default",
						Labels: map[string]string{
							string(v1beta1.DeploymentID):          deploymentId,
							string(v1beta1.FleetClusterID):        clusterId,
							string(v1beta1.FleetClusterNamespace): namespace,
						},
					},
				})
				Expect(dci.DeploymentID).To(Equal(deploymentId))
				Expect(dci.ClusterID).To(Equal(clusterId))
				Expect(dci.ClusterNamespace).To(Equal(namespace))
				Expect(dci.Name).To(Equal(depClusterName))
				Expect(err).To(BeNil())
			})
		})

		Context("the Deployment ID is missing", func() {
			It("should fail", func() {
				_, err := deploymentClusterInfo(&fleetv1alpha1.BundleDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bad-bundledeployment",
						Namespace: "default",
						Labels: map[string]string{
							string(v1beta1.FleetClusterID):        clusterId,
							string(v1beta1.FleetClusterNamespace): namespace,
						},
					},
				})
				Expect(err).ToNot(BeNil())
			})
		})

		Context("the Cluster ID is missing", func() {
			It("should fail", func() {
				_, err := deploymentClusterInfo(&fleetv1alpha1.BundleDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bad-bundledeployment",
						Namespace: "default",
						Labels: map[string]string{
							string(v1beta1.DeploymentID):          deploymentId,
							string(v1beta1.FleetClusterNamespace): namespace,
						},
					},
				})
				Expect(err).ToNot(BeNil())
			})
		})

		Context("the Cluster namespace is missing", func() {
			It("should fail", func() {
				_, err := deploymentClusterInfo(&fleetv1alpha1.BundleDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bad-bundledeployment",
						Namespace: "default",
						Labels: map[string]string{
							string(v1beta1.DeploymentID):   deploymentId,
							string(v1beta1.FleetClusterID): clusterId,
						},
					},
				})
				Expect(err).ToNot(BeNil())
			})
		})
	})

	When("adding App status to a DeploymentCluster", func() {
		It("should get appended to the list of Apps and the counts updated", func() {
			dc := &v1beta1.DeploymentCluster{
				Status: v1beta1.DeploymentClusterStatus{
					Name: clusterDisplayName,
				},
			}
			initializeStatus(dc)

			addDeploymentClusterApp(bd1, dc)
			Expect(dc.Status.Status.State).To(Equal(v1beta1.Running))
			Expect(dc.Status.Status.Summary).To(Equal(v1beta1.Summary{
				Type:    v1beta1.AppCounts,
				Total:   1,
				Running: 1,
				Down:    0,
				Unknown: 0,
			}))
			Expect(dc.Status.Apps).To(Equal([]v1beta1.App{
				{
					Name:                 appName1,
					Id:                   bundleName1,
					DeploymentGeneration: 1,
					Status: v1beta1.Status{
						State:   v1beta1.Running,
						Message: "",
						Summary: v1beta1.Summary{
							Type: v1beta1.NotUsed,
						},
					},
				},
			}))
			addDeploymentClusterApp(bd2, dc)
			Expect(dc.Status.Status.State).To(Equal(v1beta1.Down))
			Expect(dc.Status.Status.Summary).To(Equal(v1beta1.Summary{
				Type:    v1beta1.AppCounts,
				Total:   2,
				Running: 1,
				Down:    1,
				Unknown: 0,
			}))
			Expect(dc.Status.Apps).To(Equal([]v1beta1.App{
				{
					Name:                 appName1,
					Id:                   bundleName1,
					DeploymentGeneration: 1,
					Status: v1beta1.Status{
						State:   v1beta1.Running,
						Message: "",
						Summary: v1beta1.Summary{
							Type: v1beta1.NotUsed,
						},
					},
				},
				{
					Name:                 appName2,
					Id:                   bundleName2,
					DeploymentGeneration: 1,
					Status: v1beta1.Status{
						State:   v1beta1.Down,
						Message: appMessage2,
						Summary: v1beta1.Summary{
							Type: v1beta1.NotUsed,
						},
					},
				},
			}))
		})
	})

	// Create / delete BundleDeployments and ensure DeploymentCluster behavior.
	When("multiple BundleDeployments are created / deleted", func() {

		Context("and first BundleDeployment created", func() {
			It("should create the DeploymentCluster", func() {
				key := types.NamespacedName{
					Namespace: namespace,
					Name:      depClusterName,
				}
				depCluster := v1beta1.DeploymentCluster{}
				Expect(k8sClient.Get(ctx, key, &depCluster)).ShouldNot(Succeed())

				// Create Cluster
				Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
				cluster.Status = *clusterstatus
				Expect(k8sClient.Status().Update(ctx, cluster)).Should(Succeed())
				Expect(cluster.Status.State).To(Equal(v1beta1.Running))

				// Create BundleDeployment
				Expect(k8sClient.Create(ctx, bd1)).Should(Succeed())

				// Wait for DeploymentCluster to be created and its Total app count to be 1
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, &depCluster)
					if err != nil {
						return false
					}
					return (depCluster.Status.Status.Summary.Total == 1)
				}, timeout, interval).Should(BeTrue())

				// Verify DeploymentCluster status
				Expect(depCluster.Spec.DeploymentID).Should(Equal(deploymentId))
				Expect(depCluster.Spec.ClusterID).Should(Equal(clusterId))
				Expect(depCluster.Spec.Namespace).Should(Equal(namespace))
				Expect(depCluster.Status.Status.Summary.Type).Should(Equal(v1beta1.AppCounts))
			})
		})

		Context("and second BundleDeployment created", func() {
			It("should update the DeploymentCluster", func() {
				key := types.NamespacedName{
					Namespace: namespace,
					Name:      depClusterName,
				}
				depCluster := v1beta1.DeploymentCluster{}
				Expect(k8sClient.Get(ctx, key, &depCluster)).Should(Succeed())

				// Create BundleDeployment
				Expect(k8sClient.Create(ctx, bd2)).Should(Succeed())

				// Wait for DeploymentCluster to be created and its Total app count to be 2
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, &depCluster)
					if err != nil {
						return false
					}
					return (depCluster.Status.Status.Summary.Total == 2)
				}, timeout, interval).Should(BeTrue())

				// Verify DeploymentCluster status
				Expect(depCluster.Spec.DeploymentID).Should(Equal(deploymentId))
				Expect(depCluster.Spec.ClusterID).Should(Equal(clusterId))
				Expect(depCluster.Spec.Namespace).Should(Equal(namespace))
				Expect(depCluster.Status.Status.Summary.Type).Should(Equal(v1beta1.AppCounts))
			})
		})

		Context("and Cluster state updated to Unknown", func() {
			It("should update the DeploymentCluster", func() {
				clusterkey := types.NamespacedName{
					Namespace: namespace,
					Name:      clusterId,
				}

				Expect(k8sClient.Get(ctx, clusterkey, cluster)).Should(Succeed())
				Expect(cluster.Status.State).To(Equal(v1beta1.Running))

				cluster.Status.State = v1beta1.Unknown
				Expect(k8sClient.Status().Update(ctx, cluster)).Should(Succeed())

				// Wait for DeploymentCluster to be updated to Unknown state
				dckey := types.NamespacedName{
					Namespace: namespace,
					Name:      depClusterName,
				}
				depCluster := v1beta1.DeploymentCluster{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, dckey, &depCluster)
					if err != nil {
						return false
					}
					return (depCluster.Status.Status.State == v1beta1.Unknown)
				}, timeout, interval).Should(BeTrue())

				// // Verify DeploymentCluster status
				Expect(depCluster.Spec.DeploymentID).Should(Equal(deploymentId))
				Expect(depCluster.Spec.ClusterID).Should(Equal(clusterId))
				Expect(depCluster.Spec.Namespace).Should(Equal(namespace))
				Expect(depCluster.Status.Status.Summary.Type).Should(Equal(v1beta1.AppCounts))
				Expect(depCluster.Status.Status.Summary.Total).Should(Equal(0))
				Expect(depCluster.Status.Apps).Should(BeNil())

				cluster.Status.State = v1beta1.Running
				Expect(k8sClient.Status().Update(ctx, cluster)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, dckey, &depCluster)
					if err != nil {
						return false
					}
					return (depCluster.Status.Status.Summary.Total == 2)
				}, timeout, interval).Should(BeTrue())
			})
		})

		Context("and first BundleDeployment deleted", func() {
			It("should update the DeploymentCluster", func() {
				key := types.NamespacedName{
					Namespace: namespace,
					Name:      depClusterName,
				}
				depCluster := v1beta1.DeploymentCluster{}
				Expect(k8sClient.Get(ctx, key, &depCluster)).Should(Succeed())

				// Delete BundleDeployment
				Expect(k8sClient.Delete(ctx, bd1)).Should(Succeed())

				// Wait for DeploymentCluster to be created and its Total app count to be 1
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, &depCluster)
					if err != nil {
						return false
					}
					return (depCluster.Status.Status.Summary.Total == 1)
				}, timeout, interval).Should(BeTrue())

				// Verify DeploymentCluster status
				Expect(depCluster.Spec.DeploymentID).Should(Equal(deploymentId))
				Expect(depCluster.Spec.ClusterID).Should(Equal(clusterId))
				Expect(depCluster.Spec.Namespace).Should(Equal(namespace))
				Expect(depCluster.Status.Status.Summary.Type).Should(Equal(v1beta1.AppCounts))
			})
		})

		Context("and second BundleDeployment deleted", func() {
			It("should delete the DeploymentCluster", func() {
				key := types.NamespacedName{
					Namespace: namespace,
					Name:      depClusterName,
				}
				depCluster := v1beta1.DeploymentCluster{}
				Expect(k8sClient.Get(ctx, key, &depCluster)).Should(Succeed())

				// Delete BundleDeployment
				Expect(k8sClient.Delete(ctx, bd2)).Should(Succeed())

				// Wait for Deployment Cluster to be deleted
				Eventually(func() error {
					return k8sClient.Get(ctx, key, &depCluster)
				}, timeout, interval).ShouldNot(Succeed())
			})
		})
	})
})
