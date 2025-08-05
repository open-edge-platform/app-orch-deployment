// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"errors"
	"fmt"
	nbmocks "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"
	"net/http"
	"net/http/httptest"

	"github.com/bufbuild/protovalidate-go"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

const INVALID_CLUSTERID = "cluster-invalid"

var _ = Describe("Gateway gRPC Service", func() {
	var (
		deploymentServer *DeploymentSvc
		clusterInstance  *deploymentv1beta1.Cluster
		crClient         *nbmocks.FakeDeploymentV1
		protoValidator   *protovalidate.Validator
		err              error
	)

	Describe("Gateway API Cluster Service GetKubeConfig", func() {
		BeforeEach(func() {
			var clusterListSrc deploymentv1beta1.ClusterList
			setClusterListObject(&clusterListSrc)

			clusterInstance = setClusterInstance(&clusterListSrc)

			crClient = &nbmocks.FakeDeploymentV1{}

			// protovalidate Validator
			protoValidator, err = protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			deploymentServer = NewDeployment(crClient, nil, nil, nil, nil, protoValidator, nil)
		})

		It("successfully get kubeconfig", func() {
			crClient := &nbmocks.FakeDeploymentV1{}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			deploymentServer := NewDeployment(crClient, nil, kc, nil, nil, protoValidator, nil)

			crClient.On(
				"Get", context.TODO(), mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(clusterInstance, nil)

			_, err = deploymentServer.GetKubeConfig(context.TODO(), &deploymentpb.GetKubeConfigRequest{
				ClusterId: "123456",
			})

			Expect(err).ToNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("fails due to secret", func() {
			crClient := &nbmocks.FakeDeploymentV1{}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			deploymentServer := NewDeployment(crClient, nil, kc, nil, nil, protoValidator, nil)

			crClient.On(
				"Get", context.TODO(), mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(clusterInstance, nil)

			_, err = deploymentServer.GetKubeConfig(context.TODO(), &deploymentpb.GetKubeConfigRequest{
				ClusterId: "123456",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Internal))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("the server does not allow this method on the requested resource (get secrets test-kubeconfig)"))
		})

		It("fails due to get of kubeconfig", func() {
			crClient.On(
				"Get", context.TODO(), mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(clusterInstance, errors.New("mock err"))

			_, err = deploymentServer.GetKubeConfig(context.TODO(), &deploymentpb.GetKubeConfigRequest{
				ClusterId: "123456",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock err"))
		})

		It("fails due to missing cluster id in request", func() {
			_, err = deploymentServer.GetKubeConfig(context.TODO(), &deploymentpb.GetKubeConfigRequest{
				ClusterId: "",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("incomplete request"))
		})

		It("fails due to cluster id validation", func() {
			_, err = deploymentServer.GetKubeConfig(context.TODO(), &deploymentpb.GetKubeConfigRequest{
				ClusterId: "TEST",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - cluster_id: value does not " +
				"match regex pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})
	})

	Describe("Gateway API Cluster Service ListClusters", func() {
		var clusterListSrc deploymentv1beta1.ClusterList

		BeforeEach(func() {
			setClusterListObject(&clusterListSrc)

			clusterInstance = setClusterInstance(&clusterListSrc)

			crClient = &nbmocks.FakeDeploymentV1{}

			// protovalidate Validator
			protoValidator, err := protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			deploymentServer = NewDeployment(crClient, nil, nil, nil, nil, protoValidator, nil)
		})

		It("successfully list clusters", func() {
			crClient.On(
				"List", context.TODO(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.ClusterList{
				ListMeta: clusterListSrc.ListMeta,
				TypeMeta: clusterListSrc.TypeMeta,
				Items:    clusterListSrc.Items,
			}, nil).Once()

			_, err := deploymentServer.ListClusters(context.TODO(), &deploymentpb.ListClustersRequest{})

			Expect(err).ToNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("fails due to page size greater than 100", func() {
			crClient.On(
				"List", context.TODO(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.ClusterList{
				ListMeta: clusterListSrc.ListMeta,
				TypeMeta: clusterListSrc.TypeMeta,
				Items:    clusterListSrc.Items,
			}, nil).Once()

			_, err := deploymentServer.ListClusters(context.TODO(), &deploymentpb.ListClustersRequest{
				PageSize: 200,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - page_size: value must be greater " +
				"than or equal to 0 and less than or equal to 100 [int32.gte_lte]"))
		})

		It("fails due to list clusters", func() {
			crClient.On(
				"List", context.TODO(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.ClusterList{
				ListMeta: clusterListSrc.ListMeta,
				TypeMeta: clusterListSrc.TypeMeta,
				Items:    clusterListSrc.Items,
			}, errors.New("mock err")).Once()

			_, err := deploymentServer.ListClusters(context.TODO(), &deploymentpb.ListClustersRequest{})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock err"))
		})
	})

	Describe("Gateway API GetCluster", func() {
		var deploymentClusterListSrc deploymentv1beta1.DeploymentClusterList

		BeforeEach(func() {
			// protovalidate Validator
			protoValidator, err := protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			crClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(crClient, nil, nil, nil, nil, protoValidator, nil)

			// populates a mock deployment object
			setDeploymentClusterListObject(&deploymentClusterListSrc)

			crClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()
		})

		It("successfully returns the deployment cluster", func() {
			resp, err := deploymentServer.GetCluster(context.Background(), &deploymentpb.GetClusterRequest{
				ClusterId: VALID_CLUSTERID,
			})

			Expect(err).Should(Succeed())
			Expect(resp.Cluster.Name).To(Equal("test-cluster-displayname"))
		})

		It("fails due to no cluster found", func() {
			protoValidator, err := protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			crClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(crClient, nil, nil, nil, nil, protoValidator, nil)

			crClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{}, nil).Once()

			_, err = deploymentServer.GetCluster(context.Background(), &deploymentpb.GetClusterRequest{
				ClusterId: INVALID_CLUSTERID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("cluster id cluster-invalid not found"))
		})

		It("fails due to missing clusterId", func() {
			_, err := deploymentServer.GetCluster(context.Background(), &deploymentpb.GetClusterRequest{
				ClusterId: "",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("incomplete request"))
		})

		It("fails due to deploymentCluster LIST error", func() {
			protoValidator, err := protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			crClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(crClient, nil, nil, nil, nil, protoValidator, nil)

			crClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, errors.New("mock deployment list err")).Once()

			_, err = deploymentServer.GetCluster(context.Background(), &deploymentpb.GetClusterRequest{
				ClusterId: VALID_CLUSTERID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock deployment list err"))
		})
	})
})

func setClusterListObject(clusterListSrc *deploymentv1beta1.ClusterList) {
	clusterListSrc.TypeMeta.Kind = KIND_C
	clusterListSrc.TypeMeta.APIVersion = apiVersion

	clusterListSrc.ListMeta.ResourceVersion = "1"
	clusterListSrc.ListMeta.Continue = "yes"
	remainingItem := int64(10)
	clusterListSrc.ListMeta.RemainingItemCount = &remainingItem

	clusterListSrc.Items = make([]deploymentv1beta1.Cluster, 1)

	setClusterObject(&clusterListSrc.Items[0])
}

func setClusterObject(clusterSrc *deploymentv1beta1.Cluster) {
	clusterSrc.ObjectMeta.Name = "test-deployment"
	clusterSrc.ObjectMeta.GenerateName = "test-generate-name"
	clusterSrc.ObjectMeta.Namespace = VALID_PROJECT_ID
	clusterSrc.ObjectMeta.UID = types.UID(VALID_UID)
	clusterSrc.ObjectMeta.ResourceVersion = "6"
	clusterSrc.ObjectMeta.Generation = 24456

	currentTime := metav1.Now()
	clusterSrc.ObjectMeta.CreationTimestamp = currentTime
	clusterSrc.ObjectMeta.DeletionTimestamp = &currentTime

	clusterSrc.ObjectMeta.Labels = make(map[string]string)
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/name"] = "deployment"
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/instance"] = clusterSrc.ObjectMeta.Name
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/part-of"] = "app-deployment-manager"
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/managed-by"] = "kustomize"
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/created-by"] = "app-deployment-manager"

	clusterSrc.Spec.DisplayName = "test display name"
	clusterSrc.Spec.KubeConfigSecretName = "test-kubeconfig"
	// clusterSrc.Spec.Project = "app.edge-orchestrator.intel.com"

}

func setClusterInstance(clusterListSrc *deploymentv1beta1.ClusterList) *deploymentv1beta1.Cluster {
	instance := &deploymentv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterListSrc.Items[0].ObjectMeta.Name,
			Namespace: clusterListSrc.Items[0].ObjectMeta.Namespace,
			Labels:    clusterListSrc.Items[0].ObjectMeta.Labels,
			UID:       types.UID(VALID_UID),
		},
		Spec: deploymentv1beta1.ClusterSpec{
			Name:                 clusterListSrc.Items[0].Spec.Name,
			DisplayName:          clusterListSrc.Items[0].Spec.DisplayName,
			KubeConfigSecretName: clusterListSrc.Items[0].Spec.KubeConfigSecretName,
		},
	}

	return instance
}

var _ = Describe("MaxItems Validation", func() {
	var (
		deploymentServer *DeploymentSvc
		crClient         *nbmocks.FakeDeploymentV1
		protoValidator   *protovalidate.Validator
		err              error
	)

	BeforeEach(func() {
		crClient = &nbmocks.FakeDeploymentV1{}
		protoValidator, err = protovalidate.New()
		Expect(err).ToNot(HaveOccurred())
		
		deploymentServer = NewDeployment(crClient, nil, nil, nil, nil, protoValidator, nil)
	})

	Describe("Validation Constants", func() {
		It("should have correct maxItems values", func() {
			Expect(MAX_LABELS_PER_REQUEST_CLUSTERS).To(Equal(100))
			Expect(MAX_LABELS_PER_REQUEST_DEPLOYMENTS).To(Equal(20))
			Expect(MAX_CLUSTERS_RESPONSE).To(Equal(1000))
			Expect(MAX_DEPLOYMENTS_RESPONSE).To(Equal(1000))
		})
	})

	Describe("Input Validation", func() {
		Context("ListClusters labels validation", func() {
			It("should accept labels within limit", func() {
				// Test with exactly 100 labels (at limit)
				labels := make([]string, 100)
				for i := 0; i < 100; i++ {
					labels[i] = fmt.Sprintf("label%d=value%d", i, i)
				}

				crClient.On("List", mock.Anything, mock.Anything).Return(&deploymentv1beta1.ClusterList{
					Items: []deploymentv1beta1.Cluster{},
				}, nil)

				req := &deploymentpb.ListClustersRequest{Labels: labels}
				_, err := deploymentServer.ListClusters(context.Background(), req)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should reject labels over limit", func() {
				// Test with 101 labels (over limit)
				labels := make([]string, 101)
				for i := 0; i < 101; i++ {
					labels[i] = fmt.Sprintf("label%d=value%d", i, i)
				}

				req := &deploymentpb.ListClustersRequest{Labels: labels}
				_, err := deploymentServer.ListClusters(context.Background(), req)
				
				Expect(err).To(HaveOccurred())
				st, ok := status.FromError(err)
				Expect(ok).To(BeTrue())
				Expect(st.Code()).To(Equal(codes.InvalidArgument))
				Expect(st.Message()).To(ContainSubstring("labels array exceeds maximum size of 100 items"))
			})
		})

		Context("Deployment service labels validation", func() {
			It("should accept deployment labels within limit", func() {
				// Test with exactly 20 labels (at limit)
				labels := make([]string, 20)
				for i := 0; i < 20; i++ {
					labels[i] = fmt.Sprintf("label%d=value%d", i, i)
				}

				crClient.On("List", mock.Anything, mock.Anything).Return(&deploymentv1beta1.DeploymentList{
					Items: []deploymentv1beta1.Deployment{},
				}, nil)

				req := &deploymentpb.ListDeploymentsRequest{Labels: labels}
				_, err := deploymentServer.ListDeployments(context.Background(), req)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should reject deployment labels over limit", func() {
				// Test with 21 labels (over limit)
				labels := make([]string, 21)
				for i := 0; i < 21; i++ {
					labels[i] = fmt.Sprintf("label%d=value%d", i, i)
				}

				req := &deploymentpb.ListDeploymentsRequest{Labels: labels}
				_, err := deploymentServer.ListDeployments(context.Background(), req)
				
				Expect(err).To(HaveOccurred())
				st, ok := status.FromError(err)
				Expect(ok).To(BeTrue())
				Expect(st.Code()).To(Equal(codes.InvalidArgument))
				Expect(st.Message()).To(ContainSubstring("labels array exceeds maximum size of 20 items"))
			})
		})
	})

	Describe("Response Array Limits", func() {
		Context("Validation Constants", func() {
			It("should have correct boundary values", func() {
				// Test that our limits are properly configured
				Expect(MAX_LABELS_PER_REQUEST_CLUSTERS).To(Equal(100))
				Expect(MAX_LABELS_PER_REQUEST_DEPLOYMENTS).To(Equal(20))
				Expect(MAX_CLUSTERS_RESPONSE).To(Equal(1000))
				Expect(MAX_DEPLOYMENTS_RESPONSE).To(Equal(1000))
			})
		})
	})
})
