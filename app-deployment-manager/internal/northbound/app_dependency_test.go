// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("App Dependency", func() {
	var (
		deploymentListSrc0 deploymentv1beta1.DeploymentList
		err                error
	)

	Describe("getTargetDeploymentsForDeleteDeployment", func() {
		BeforeEach(func() {
			var clusterListSrc deploymentv1beta1.ClusterList
			setClusterListObject(&clusterListSrc)

			// populates a mock deployment object
			setDeploymentListObjects(&deploymentListSrc0)
		})

		It("successfully get target deployments for delete deployment", func() {
			d := setDeployment()

			targetList := make(map[string]*deploymentv1beta1.Deployment)

			allDeploymentMap := make(map[string]*deploymentv1beta1.Deployment)

			for i, depl := range deploymentListSrc0.Items {
				allDeploymentMap[depl.Name] = &deploymentListSrc0.Items[i]
			}

			err = getTargetDeploymentsForDeleteDeployment(d.Name, targetList, allDeploymentMap, 0)

			Expect(err).ToNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("successfully validate target deployments", func() {
			targetList := make(map[string]*deploymentv1beta1.Deployment)

			out := validateTargetDeployments(targetList)

			Expect(out).To(BeTrue())
		})

		It("successfully get deployment with deployment package", func() {
			targetList := make(map[string]*Deployment)

			dpID := "123456"

			out := getDeploymentWithDeploymentPackage(dpID, targetList)

			Expect(out).To(BeNil())
		})

		It("successfully output false to contain target cluster", func() {
			inList := []map[string]string{{"hello": "world"}}

			inMap := make(map[string]string)
			inMap["foo"] = "woo"

			out := containTargetCluster(inList, inMap)
			Expect(out).To(BeFalse())
		})

		It("successfully output true to contain target cluster", func() {
			inList := []map[string]string{{"hello": "world"}}

			inMap := make(map[string]string)
			inMap["hello"] = "world"

			out := containTargetCluster(inList, inMap)
			Expect(out).To(BeTrue())
		})

		It("successfully get target labels from cluster map", func() {
			targetClustersList := &deploymentpb.TargetClusters{
				AppName: "test-appname",
				Labels:  map[string]string{"test": "foo"},
			}

			out := getTargetClusterMap(targetClustersList)
			Expect(out["test"]).To(Equal("foo"))
		})

		It("successfully get target cluster id from cluster map", func() {
			targetClustersList := &deploymentpb.TargetClusters{
				AppName:   "test-appname",
				ClusterId: "123456",
			}

			out := getTargetClusterMap(targetClustersList)
			Expect(out[string(deploymentv1beta1.ClusterName)]).To(Equal("123456"))
		})
	})
})
