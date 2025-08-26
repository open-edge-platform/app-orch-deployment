// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"google.golang.org/grpc/status"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

var _ = Describe("Data Selection", func() {
	var (
		deploymentListSrc0 deploymentv1beta1.DeploymentList
		deploy             *deploymentpb.Deployment
		deployList         []*deploymentpb.Deployment
	)

	Describe("Test selectDeployments", func() {
		BeforeEach(func() {
			setDeploymentListObject(&deploymentListSrc0)
			deploy = getDeployInstance(&deploymentListSrc0)
		})

		It("successfully select deployments", func() {
			deployList = append(deployList, deploy)
			var listDeployReq = deploymentpb.ListDeploymentsRequest{
				PageSize: 2,
				Offset:   3,
				OrderBy:  "name asc, id asc",
				Filter:   "name=abc",
			}

			_, err := selectDeployments(&listDeployReq, deployList)
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully select deployments when filter is empty", func() {
			deployList = append(deployList, deploy)
			var listDeployReq = deploymentpb.ListDeploymentsRequest{
				PageSize: 2,
				Offset:   3,
				OrderBy:  "name asc, id asc",
				Filter:   "",
			}

			_, err := selectDeployments(&listDeployReq, deployList)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails due to pagesize exceeds maximum", func() {
			deployList = append(deployList, deploy)
			var listDeployReq = deploymentpb.ListDeploymentsRequest{
				PageSize: 101, // exceeds maximum of 100
				Offset:   0,
			}

			_, err := selectDeployments(&listDeployReq, deployList)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("pagesize parameter must be lte 100"))
		})

		It("fails due to parse order by", func() {
			deployList = append(deployList, deploy)
			var listDeployReq = deploymentpb.ListDeploymentsRequest{
				OrderBy: "name test, id asc",
			}

			_, err := selectDeployments(&listDeployReq, deployList)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("invalid order direction; must be 'asc' or 'desc'"))
		})

		It("fails due to parse filter by", func() {
			deployList = append(deployList, deploy)
			var listDeployReq = deploymentpb.ListDeploymentsRequest{
				Filter: "name AND abc",
			}

			_, err := selectDeployments(&listDeployReq, deployList)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("invalid filter request"))
		})
	})

	Describe("Test selectClusters", func() {
		var createTimePbUnix *timestamppb.Timestamp
		var testLabels = make(map[string]string)

		BeforeEach(func() {
			currentTime := metav1.Now()

			// Convert metav1.Time to protobuf time and return secs
			setPbTime := currentTime.ProtoTime()

			// Return Timestamp from the provided time.Time in unix
			createTimePbUnix = timestamppb.New(time.Unix(int64(setPbTime.Seconds), 0))

			testLabels["hellow"] = "world"
		})

		It("successfully select clusters", func() {
			var clusterListReq = deploymentpb.ListClustersRequest{
				OrderBy:  "name asc, id asc",
				Labels:   []string{"test-labels"},
				PageSize: 2,
				Offset:   3,
				Filter:   "name=abc",
			}

			clusterLists := make([]*deploymentpb.ClusterInfo, 1)
			clusterLists[0] = &deploymentpb.ClusterInfo{
				Id:         "123456-123456",
				Labels:     testLabels,
				CreateTime: createTimePbUnix,
				Name:       "test-name",
			}

			_, err := selectClusters(&clusterListReq, clusterLists)

			Expect(err).ToNot(HaveOccurred())
		})

		It("fails due to pagesize exceeds maximum", func() {
			var clusterListReq = deploymentpb.ListClustersRequest{
				PageSize: 101, // exceeds maximum of 100
				Offset:   0,
			}

			clusterLists := make([]*deploymentpb.ClusterInfo, 1)
			clusterLists[0] = &deploymentpb.ClusterInfo{
				Id:         "123456-123456",
				Labels:     testLabels,
				CreateTime: createTimePbUnix,
				Name:       "test-name",
			}

			_, err := selectClusters(&clusterListReq, clusterLists)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("pagesize parameter must be lte 100"))
		})

		It("fails due to parse order by", func() {
			var clusterListReq = deploymentpb.ListClustersRequest{
				OrderBy: "name test, id asc",
			}

			clusterLists := make([]*deploymentpb.ClusterInfo, 1)
			clusterLists[0] = &deploymentpb.ClusterInfo{
				Id:         "123456-123456",
				Labels:     testLabels,
				CreateTime: createTimePbUnix,
				Name:       "test-name",
			}

			_, err := selectClusters(&clusterListReq, clusterLists)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("invalid order direction; must be 'asc' or 'desc'"))
		})

		It("fails due to parse filter by", func() {
			var clusterListReq = deploymentpb.ListClustersRequest{
				Filter: "name AND abc",
			}

			clusterLists := make([]*deploymentpb.ClusterInfo, 1)
			clusterLists[0] = &deploymentpb.ClusterInfo{
				Id:         "123456-123456",
				Labels:     testLabels,
				CreateTime: createTimePbUnix,
				Name:       "test-name",
			}

			_, err := selectClusters(&clusterListReq, clusterLists)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("invalid filter request"))
		})

		It("successfully select cluster when filter is empty", func() {
			var clusterListReq = deploymentpb.ListClustersRequest{
				OrderBy:  "name asc, id asc",
				Labels:   []string{"test-labels"},
				PageSize: 2,
				Offset:   3,
				Filter:   "",
			}

			clusterLists := make([]*deploymentpb.ClusterInfo, 1)
			clusterLists[0] = &deploymentpb.ClusterInfo{
				Id:         "123456-123456",
				Labels:     testLabels,
				CreateTime: createTimePbUnix,
				Name:       "test-name",
			}

			_, err := selectClusters(&clusterListReq, clusterLists)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Test selectClustersPerDeployment", func() {
		var clusterLists = make([]*deploymentpb.Cluster, 1)

		BeforeEach(func() {
			clusterLists[0] = getClusterInstance()
		})

		It("successfully select clusters per deployment", func() {
			var clusterListReq = deploymentpb.ListDeploymentClustersRequest{
				OrderBy:  "name asc, id asc",
				DeplId:   "123456-123456",
				PageSize: 2,
				Offset:   3,
				Filter:   "name=abc",
			}

			_, err := selectClustersPerDeployment(&clusterListReq, clusterLists)

			Expect(err).ToNot(HaveOccurred())
		})

		It("fails due to pagesize exceeds maximum", func() {
			var clusterListReq = deploymentpb.ListDeploymentClustersRequest{
				PageSize: 101, // exceeds maximum of 100
				Offset:   0,
			}

			_, err := selectClustersPerDeployment(&clusterListReq, clusterLists)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("pagesize parameter must be lte 100"))
		})

		It("fails due to parse order by", func() {
			var clusterListReq = deploymentpb.ListDeploymentClustersRequest{
				OrderBy: "name test, id asc",
			}

			_, err := selectClustersPerDeployment(&clusterListReq, clusterLists)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("invalid order direction; must be 'asc' or 'desc'"))
		})

		It("fails due to parse filter by", func() {
			var clusterListReq = deploymentpb.ListDeploymentClustersRequest{
				Filter: "name AND abc",
			}

			_, err := selectClustersPerDeployment(&clusterListReq, clusterLists)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("invalid filter request"))
		})

		It("successfully select cluster when filter is empty", func() {
			var clusterListReq = deploymentpb.ListDeploymentClustersRequest{
				OrderBy:  "name asc, id asc",
				DeplId:   "123456-123456",
				PageSize: 2,
				Offset:   3,
				Filter:   "",
			}

			_, err := selectClustersPerDeployment(&clusterListReq, clusterLists)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Test newPaginationQuery", func() {
		var clusterLists = make([]*deploymentpb.Cluster, 1)

		BeforeEach(func() {
			clusterLists[0] = getClusterInstance()
		})

		It("successfully get pagination query", func() {
			offset := 1
			pageSize := 0

			v := newPaginationQuery(uint32(pageSize), uint32(offset))
			Expect(v.PageSize).To(Equal(10))
			Expect(v.OffSet).To(Equal(1))
		})

	})
})
