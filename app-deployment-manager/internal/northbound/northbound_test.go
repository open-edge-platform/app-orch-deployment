// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"os"
	"testing"

	nbmocks "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"

	"net/http"
	"net/http/httptest"

	"google.golang.org/grpc/metadata"

	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	mockerymock "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient/mockery"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const apiVersion = "v1.1"
const VALID_UID = "123456-123456-123456"
const VALID_UID_DC = "dc-789789-789789-789789"
const VALID_CLUSTERID = "cluster-01234567"
const INVALID_UID = "023456-023456-023456"
const KIND = "deployments"
const KIND_DC = "deploymentclusters"
const KIND_C = "clusters"
const VALID_CLUSTER_ID = "cluster-0123456789"
const VALID_PROJECT_ID = "0000-1111-2222-3333-4444"

var origMatchUIDDeploymentFn = matchUIDDeployment

func TestGateway(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gateway Suite")
}

type DeployServer struct {
	deploymentServer *DeploymentSvc
	k8sClient        *nbmocks.FakeDeploymentV1
	vaultAuthMock    *nbmocks.VaultAuth
	opaMock          *openpolicyagent.MockClientWithResponsesInterface
	catalogClient    *mockerymock.MockeryCatalogClient
	kc               *kubernetes.Clientset
	ctx              context.Context
}

var _ = Describe("Gateway gRPC Service", func() {
	var (
		deploymentServer         *DeploymentSvc
		deploymentListSrc        deploymentv1beta1.DeploymentList
		k8sClient                *nbmocks.FakeDeploymentV1
		deployInstance           *deploymentv1beta1.Deployment
		deployInstanceResp       *deploymentpb.Deployment
		mockController           *gomock.Controller
		matchingLabelList        []string
		deploymentClusterListSrc deploymentv1beta1.DeploymentClusterList
		t                        TestReporter
		s                        DeployServer
		ts                       *httptest.Server
		err                      error
		ctx                      context.Context
	)

	Describe("Gateway API Auth Check", func() {
		BeforeEach(func() {
			k8sClient = &nbmocks.FakeDeploymentV1{}
			mockController = gomock.NewController(t)
			opaMock := openpolicyagent.NewMockClientWithResponsesInterface(mockController)

			result := openpolicyagent.OpaResponse_Result{}
			err := result.FromOpaResponseResult1(false)
			Expect(err).ToNot(HaveOccurred())

			opaMock.EXPECT().PostV1DataPackageRuleWithBodyWithResponse(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any()).Return(
				&openpolicyagent.PostV1DataPackageRuleResponse{
					JSON200: &openpolicyagent.OpaResponse{
						DecisionId: nil,
						Metrics:    nil,
						Result:     result,
					},
				}, nil,
			).AnyTimes()

			// protovalidate Validator

			deploymentServer = NewDeployment(k8sClient, opaMock, nil, nil, nil, nil)

			md := metadata.Pairs("foo", "bar")
			ctx = metadata.NewIncomingContext(context.Background(), md)

			setDeploymentListObjects(&deploymentListSrc)
			deployInstanceResp = getDeployInstance(&deploymentListSrc)
		})

		It("GetDeploymentsStatus: fails due to access denied", func() {
			_, err := deploymentServer.GetDeploymentsStatus(ctx, &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.PermissionDenied))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("cannot get status of deployments: access denied (3)"))
		})

		It("CreateDeployment: fails due to access denied", func() {
			_, err := deploymentServer.CreateDeployment(ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.PermissionDenied))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("cannot create deployment: access denied (3)"))
		})

		It("GetDeployment: fails due to access denied", func() {
			_, err := deploymentServer.GetDeployment(ctx, &deploymentpb.GetDeploymentRequest{
				DeplId: VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.PermissionDenied))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("cannot get deployment: access denied (3)"))
		})

		It("ListDeployments: fails due to access denied", func() {
			_, err := deploymentServer.ListDeployments(ctx, &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.PermissionDenied))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("cannot list deployments: access denied (3)"))
		})

		It("DeleteDeployment: fails due to access denied", func() {
			_, err := deploymentServer.DeleteDeployment(ctx, &deploymentpb.DeleteDeploymentRequest{
				DeplId: VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.PermissionDenied))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("cannot delete deployment: access denied (3)"))
		})

		It("UpdateDeployment: fails due to access denied", func() {
			_, err := deploymentServer.UpdateDeployment(ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.PermissionDenied))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("cannot update deployment: access denied (3)"))
		})

		It("ListDeploymentClusters: fails due to access denied", func() {
			_, err := deploymentServer.ListDeploymentClusters(ctx, &deploymentpb.ListDeploymentClustersRequest{
				DeplId: VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.PermissionDenied))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("cannot get deployment clusters: access denied (3)"))
		})
	})

	Describe("Gateway API ListDeploymentClusters", func() {
		BeforeEach(func() {
			// protovalidate Validator

			k8sClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			// populates a mock deployment object
			setDeploymentListObjects(&deploymentListSrc)
			setDeploymentClusterListObject(&deploymentClusterListSrc)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()
		})

		It("successfully returns list of deployment with 3 clusters", func() {
			resp, err := deploymentServer.ListDeploymentClusters(context.Background(), &deploymentpb.ListDeploymentClustersRequest{
				DeplId: VALID_UID,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Clusters)).To(Equal(3))
		})

		It("fails due to missing deplId", func() {
			_, err := deploymentServer.ListDeploymentClusters(context.Background(), &deploymentpb.ListDeploymentClustersRequest{
				DeplId: "",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("incomplete request"))
		})

		It("fails due to deployment LIST error", func() {
			k8sClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, errors.New("mock deployment list err")).Once()

			_, err := deploymentServer.ListDeploymentClusters(context.Background(), &deploymentpb.ListDeploymentClustersRequest{
				DeplId: VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock deployment list err"))
		})

		It("fails due to deployment cluster LIST error", func() {
			k8sClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, errors.New("mock deployment cluster list err")).Once()

			_, err := deploymentServer.ListDeploymentClusters(context.Background(), &deploymentpb.ListDeploymentClustersRequest{
				DeplId: VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock deployment cluster list err"))
		})

		It("fails due to deployment not found", func() {
			k8sClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			deploymentListSrc.Items[0].ObjectMeta.Name = ""

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			_, err := deploymentServer.ListDeploymentClusters(context.Background(), &deploymentpb.ListDeploymentClustersRequest{
				DeplId: VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("deployment id 123456-123456-123456 not found"))
		})
	})

	Describe("Gateway API Deployment Status", func() {
		BeforeEach(func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			s.kc = mockK8Client(ts.URL)

			// protovalidate Validator

			s.k8sClient = &nbmocks.FakeDeploymentV1{}
			s.deploymentServer = NewDeployment(s.k8sClient, nil, s.kc, nil, nil, nil)

			// populates a mock deployment object
			setDeploymentListObjects(&deploymentListSrc)
			setDeploymentClusterListObject(&deploymentClusterListSrc)

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()
		})

		It("successfully returns all deployments status with no filter labels", func() {
			deploymentListSrc.Items[0].Status.Summary.Total = 2
			deploymentListSrc.Items[0].Status.Summary.Running = 2
			deploymentListSrc.Items[0].Status.Summary.Down = 0
			deploymentListSrc.Items[0].Status.Summary.Unknown = 0
			deploymentListSrc.Items[0].Status.State = "Running"

			deploymentListSrc.Items[1].Status.Summary.Total = 3
			deploymentListSrc.Items[1].Status.Summary.Running = 2
			deploymentListSrc.Items[1].Status.Summary.Down = 1
			deploymentListSrc.Items[1].Status.Summary.Unknown = 0
			deploymentListSrc.Items[1].Status.State = "Down"

			deploymentListSrc.Items[2].Status.Summary.Total = 4
			deploymentListSrc.Items[2].Status.Summary.Running = 3
			deploymentListSrc.Items[2].Status.Summary.Down = 1
			deploymentListSrc.Items[2].Status.Summary.Unknown = 0
			deploymentListSrc.Items[2].Status.State = "Down"

			resp, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(9)))
			Expect((resp.Running)).To(Equal(int32(7)))
			Expect((resp.Down)).To(Equal(int32(2)))
			Expect((resp.Unknown)).To(Equal(int32(0)))
		})

		It("successfully returns all deployments status with no filter labels 1 deployment", func() {
			deploymentListSrc.Items[0].Status.Summary.Total = 2
			deploymentListSrc.Items[0].Status.Summary.Running = 1
			deploymentListSrc.Items[0].Status.Summary.Down = 0
			deploymentListSrc.Items[0].Status.Summary.Unknown = 1
			deploymentListSrc.Items[0].Status.State = "Unknown"

			deploymentListSrc.Items[1].Status.Summary.Total = 1
			deploymentListSrc.Items[1].Status.Summary.Running = 1
			deploymentListSrc.Items[1].Status.Summary.Down = 0
			deploymentListSrc.Items[1].Status.Summary.Unknown = 0
			deploymentListSrc.Items[1].Status.State = "Running"

			deploymentListSrc.Items[2].Status.Summary.Total = 0
			deploymentListSrc.Items[2].Status.Summary.Running = 0
			deploymentListSrc.Items[2].Status.Summary.Down = 0
			deploymentListSrc.Items[2].Status.Summary.Unknown = 0
			deploymentListSrc.Items[2].Status.State = "Running"

			resp, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(3)))
			Expect((resp.Running)).To(Equal(int32(2)))
			Expect((resp.Down)).To(Equal(int32(0)))
			Expect((resp.Unknown)).To(Equal(int32(1)))

			deploymentListSrc.Items[2].Status.Summary.Total = 1
			deploymentListSrc.Items[2].Status.Summary.Running = 0
			deploymentListSrc.Items[2].Status.Summary.Down = 1
			deploymentListSrc.Items[2].Status.Summary.Unknown = 0
			deploymentListSrc.Items[2].Status.State = "Deploying"

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			resp, err = s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(4)))
			Expect((resp.Running)).To(Equal(int32(2)))
			Expect((resp.Down)).To(Equal(int32(0)))
			Expect((resp.Unknown)).To(Equal(int32(1)))
			Expect((resp.Deploying)).To(Equal(int32(1)))

			deploymentListSrc.Items[2].Status.Summary.Total = 1
			deploymentListSrc.Items[2].Status.Summary.Running = 0
			deploymentListSrc.Items[2].Status.Summary.Down = 1
			deploymentListSrc.Items[2].Status.Summary.Unknown = 0
			deploymentListSrc.Items[2].Status.State = "Updating"

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			resp, err = s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(4)))
			Expect((resp.Running)).To(Equal(int32(2)))
			Expect((resp.Down)).To(Equal(int32(0)))
			Expect((resp.Unknown)).To(Equal(int32(1)))
			Expect((resp.Updating)).To(Equal(int32(1)))

			deploymentListSrc.Items[2].Status.Summary.Total = 1
			deploymentListSrc.Items[2].Status.Summary.Running = 0
			deploymentListSrc.Items[2].Status.Summary.Down = 1
			deploymentListSrc.Items[2].Status.Summary.Unknown = 0
			deploymentListSrc.Items[2].Status.State = "Terminating"

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			resp, err = s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(4)))
			Expect((resp.Running)).To(Equal(int32(2)))
			Expect((resp.Down)).To(Equal(int32(0)))
			Expect((resp.Unknown)).To(Equal(int32(1)))
			Expect((resp.Terminating)).To(Equal(int32(1)))
		})

		It("fails due to deployment LIST error", func() {
			defer ts.Close()

			k8sClient := &nbmocks.FakeDeploymentV1{}
			deploymentServer := NewDeployment(k8sClient, nil, s.kc, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, errors.New("mock err")).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			_, err := deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})
			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock err"))
		})

		It("fails due to deployment cluster LIST error", func() {
			defer ts.Close()

			k8sClient := &nbmocks.FakeDeploymentV1{}
			deploymentServer := NewDeployment(k8sClient, nil, s.kc, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, errors.New("mock err deployment cluster")).Once()

			_, err := deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock err deployment cluster"))
		})

	})

	Describe("Gateway API Deployment Status Filter", func() {
		var doesMatch = make(map[string]string)
		var doesNotMatch = make(map[string]string)

		BeforeEach(func() {
			s.k8sClient = &nbmocks.FakeDeploymentV1{}

			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			s.kc = mockK8Client(ts.URL)

			// protovalidate Validator

			s.deploymentServer = NewDeployment(s.k8sClient, nil, s.kc, nil, nil, nil)

			// populates a mock deployment object
			setDeploymentListObjects(&deploymentListSrc)
			setDeploymentClusterListObject(&deploymentClusterListSrc)

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			matchingLabelList = append(matchingLabelList, "hello=world")
			matchingLabelList = append(matchingLabelList, "test=foo")
			matchingLabelList = append(matchingLabelList, "mock=go")
		})

		It("Filter: successfully returns all deployments status with matching filter labels", func() {
			deploymentListSrc.Items[0].Status.Summary.Total = 2
			deploymentListSrc.Items[0].Status.Summary.Running = 1
			deploymentListSrc.Items[0].Status.Summary.Down = 1
			deploymentListSrc.Items[0].Status.Summary.Unknown = 0
			deploymentListSrc.Items[0].Status.State = "Running"

			deploymentListSrc.Items[1].Status.Summary.Total = 3
			deploymentListSrc.Items[1].Status.Summary.Running = 2
			deploymentListSrc.Items[1].Status.Summary.Down = 1
			deploymentListSrc.Items[1].Status.Summary.Unknown = 0
			deploymentListSrc.Items[1].Status.State = "Running"

			deploymentListSrc.Items[2].Status.Summary.Total = 4
			deploymentListSrc.Items[2].Status.Summary.Running = 3
			deploymentListSrc.Items[2].Status.Summary.Down = 1
			deploymentListSrc.Items[2].Status.Summary.Unknown = 0
			deploymentListSrc.Items[2].Status.State = "Running"

			resp, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(9)))
			Expect((resp.Running)).To(Equal(int32(6)))
			Expect((resp.Down)).To(Equal(int32(3)))
			Expect((resp.Unknown)).To(Equal(int32(0)))
		})

		It("Filter: successfully returns 1 out of 3 deployments status in random order due matching target label", func() {
			doesMatch["hello"] = "world"
			doesNotMatch["foo"] = "world"

			for i := 0; i < 3; i++ {
				// 1st and 3rd deployments don't match
				currLabel := []map[string]string{doesNotMatch}
				if i == 1 {
					currLabel = []map[string]string{doesMatch}
				}

				deploymentListSrc.Items[i].Name = fmt.Sprintf("test%d", i)
				deploymentListSrc.Items[i].UID = types.UID(fmt.Sprintf("132456-56123-%d", i))
				deploymentListSrc.Items[i].Spec.Applications[0].Targets = currLabel

				deploymentListSrc.Items[i].Status.Summary.Total = i + 2
				deploymentListSrc.Items[i].Status.Summary.Running = i + 1
				deploymentListSrc.Items[i].Status.Summary.Down = 1
				deploymentListSrc.Items[i].Status.State = "Running"
			}

			resp, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(3)))
			Expect((resp.Running)).To(Equal(int32(2)))
			Expect((resp.Down)).To(Equal(int32(1)))
		})

		It("Filter: successfully returns of 2 out of 3 deployments status in random order due matching target label", func() {
			doesMatch["hello"] = "world"
			doesMatch["not"] = "valid"
			doesNotMatch["foo"] = "world"

			for i := 0; i < 3; i++ {
				// 2nd deployment doesn't match
				currLabel := []map[string]string{doesNotMatch}
				if i == 0 || i == 2 {
					currLabel = []map[string]string{doesMatch}
				}

				deploymentListSrc.Items[i].Name = fmt.Sprintf("test%d", i)
				deploymentListSrc.Items[i].Spec.Applications[0].Targets = currLabel

				deploymentListSrc.Items[i].Status.Summary.Total = i + 2
				deploymentListSrc.Items[i].Status.Summary.Running = i + 1
				deploymentListSrc.Items[i].Status.Summary.Down = 1
				deploymentListSrc.Items[i].Status.State = "Running"
			}

			resp, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(6)))
			Expect((resp.Running)).To(Equal(int32(4)))
			Expect((resp.Down)).To(Equal(int32(2)))
		})

		It("Filter: successfully returns 2 match label out of 3 deployments status with some labels not matching", func() {
			doesMatch["hello"] = "world"
			doesMatch["not"] = "valid"
			doesNotMatch["foo"] = "world"

			for i := 0; i < 3; i++ {
				// 3rd deployment doesn't match
				currLabel := []map[string]string{doesNotMatch}
				if i == 0 || i == 1 {
					currLabel = []map[string]string{doesMatch}
				}

				deploymentListSrc.Items[i].Name = fmt.Sprintf("test%d", i)
				deploymentListSrc.Items[i].Spec.Applications[0].Targets = currLabel

				deploymentListSrc.Items[i].Status.Summary.Total = i + 2
				deploymentListSrc.Items[i].Status.Summary.Running = i + 1
				deploymentListSrc.Items[i].Status.Summary.Down = 1
				deploymentListSrc.Items[i].Status.State = "Running"
			}

			resp, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(5)))
			Expect((resp.Running)).To(Equal(int32(3)))
			Expect((resp.Down)).To(Equal(int32(2)))
		})

		It("Filter: successfully returns 0 for all deployments status due no matching target label", func() {
			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "tester=foo")

			resp, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect((resp.Total)).To(Equal(int32(0)))
			Expect((resp.Running)).To(Equal(int32(0)))
			Expect((resp.Down)).To(Equal(int32(0)))
		})

		It("Filter: fails due to invalid pattern uppercase char in target label", func() {
			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "tesTer=foo")

			_, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - labels[0]: value does not match regex " +
				"pattern `(^$)|^[a-z0-9]([-_.=,a-z0-9]{0,198}[a-z0-9])?$` [string.pattern]"))
		})

		It("Filter: fails due to invalid pattern non alphanumeric in target label", func() {
			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "tes?er=foo")

			_, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - labels[0]: value does not match regex " +
				"pattern `(^$)|^[a-z0-9]([-_.=,a-z0-9]{0,198}[a-z0-9])?$` [string.pattern]"))
		})

		It("Filter: fails due to invalid pattern whitespace in target label", func() {
			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "tester=foo ")

			_, err := s.deploymentServer.GetDeploymentsStatus(context.Background(), &deploymentpb.GetDeploymentsStatusRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - labels[0]: value does not match regex " +
				"pattern `(^$)|^[a-z0-9]([-_.=,a-z0-9]{0,198}[a-z0-9])?$` [string.pattern]"))
		})
	})

	Describe("Gateway API List", func() {
		BeforeEach(func() {
			s.k8sClient = &nbmocks.FakeDeploymentV1{}

			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			// defer ts.Close()

			s.kc = mockK8Client(ts.URL)

			// protovalidate Validator

			s.deploymentServer = NewDeployment(s.k8sClient, nil, s.kc, nil, nil, nil)

			// populates a mock deployment object
			setDeploymentListObject(&deploymentListSrc)
			setDeploymentClusterListObject(&deploymentClusterListSrc)

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			// clear list
			matchingLabelList = matchingLabelList[:0]
		})

		It("successfully returns a populated list with no filter labels", func() {
			resp, err := s.deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Deployments)).To(Equal(1))
			Expect(resp.Deployments[0].Name).To(Equal("test-deployment"))
		})

		It("successfully returns an empty list with no filter labels", func() {
			k8sClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			resp, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Deployments)).To(Equal(0))
		})

		It("fails due to PageSize is over 100", func() {
			_, err := s.deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				PageSize: 200,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - page_size: value must be greater than " +
				"or equal to 0 and less than or equal to 100 [uint32.gte_lte]"))
		})

		It("fails due deployment LIST error", func() {
			defer ts.Close()

			k8sClient = &nbmocks.FakeDeploymentV1{}

			deploymentServer = NewDeployment(k8sClient, nil, s.kc, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, errors.New("mock deployment err")).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			_, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock deployment err"))
		})
	})

	Describe("Gateway API Filter List", func() {
		BeforeEach(func() {
			k8sClient = &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			// populates a mock deployment object
			setDeploymentListObject(&deploymentListSrc)

			deploymentListSrc.Items[0].Spec.DeploymentType = deploymentv1beta1.AutoScaling

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			matchingLabelList = append(matchingLabelList, "hello=world")
			matchingLabelList = append(matchingLabelList, "test=foo")
			matchingLabelList = append(matchingLabelList, "mock=go")
		})

		It("Filter: successfully returns a list with matching target label", func() {
			k8sClient := &nbmocks.FakeDeploymentV1{}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)
			deploymentServer := NewDeployment(k8sClient, nil, kc, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "test=foo")
			resp, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Deployments)).To(Equal(1))
			Expect(resp.Deployments[0].Name).To(Equal("test-deployment"))
		})

		It("Filter: successfully returns an empty list due no matching target label", func() {
			k8sClient := &nbmocks.FakeDeploymentV1{}
			deploymentServer := NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "tester=foo")

			resp, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Deployments)).To(Equal(0))
		})

		It("Filter: successfully returns a list of 1 out of 3 in random order due matching target label", func() {
			// populates a mock deployment object
			setDeploymentListObjects(&deploymentListSrc)

			deploymentListSrc.Items[0].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[1].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[2].Spec.DeploymentType = deploymentv1beta1.AutoScaling

			k8sClient := &nbmocks.FakeDeploymentV1{}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)
			deploymentServer := NewDeployment(k8sClient, nil, kc, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			var doesMatch = make(map[string]string)
			doesMatch["hello"] = "world"

			var doesNotMatch = make(map[string]string)
			doesNotMatch["foo"] = "world"

			for i := 0; i < 3; i++ {
				// 1st and 3rd deployments don't match
				currLabel := []map[string]string{doesNotMatch}
				if i == 1 {
					currLabel = []map[string]string{doesMatch}
				}

				deploymentListSrc.Items[i].Name = fmt.Sprintf("test%d", i)
				deploymentListSrc.Items[i].Spec.Applications[0].Targets = currLabel
			}

			resp, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Deployments)).To(Equal(1))
			Expect(resp.Deployments[0].Name).To(Equal("test1"))
		})

		It("Filter: successfully returns a list of 2 out of 3 in random order due matching target label", func() {
			// populates a mock deployment object
			setDeploymentListObjects(&deploymentListSrc)

			deploymentListSrc.Items[0].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[1].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[2].Spec.DeploymentType = deploymentv1beta1.AutoScaling

			k8sClient := &nbmocks.FakeDeploymentV1{}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kClient := mockK8Client(ts.URL)
			deploymentServer := NewDeployment(k8sClient, nil, kClient, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			var doesMatch = make(map[string]string)
			doesMatch["hello"] = "world"
			doesMatch["not"] = "valid"

			var doesNotMatch = make(map[string]string)
			doesNotMatch["foo"] = "world"

			for i := 0; i < 3; i++ {
				// 2nd deployment doesn't match
				currLabel := []map[string]string{doesNotMatch}
				if i == 0 || i == 2 {
					currLabel = []map[string]string{doesMatch}
				}

				deploymentListSrc.Items[i].Name = fmt.Sprintf("test%d", i)
				deploymentListSrc.Items[i].Spec.Applications[0].Targets = currLabel
			}

			resp, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Deployments)).To(Equal(2))
			Expect(resp.Deployments[0].Name).To(Equal("test0"))
			Expect(resp.Deployments[1].Name).To(Equal("test2"))
		})

		It("Filter: successfully returns a list of 2 out of 3 due matching target label", func() {
			// populates a mock deployment object
			setDeploymentListObjects(&deploymentListSrc)

			deploymentListSrc.Items[0].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[1].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[2].Spec.DeploymentType = deploymentv1beta1.AutoScaling

			k8sClient := &nbmocks.FakeDeploymentV1{}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kClient := mockK8Client(ts.URL)
			deploymentServer := NewDeployment(k8sClient, nil, kClient, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			var doesMatch = make(map[string]string)
			doesMatch["hello"] = "world"

			var doesNotMatch = make(map[string]string)
			doesNotMatch["foo"] = "world"

			for i := 0; i < 3; i++ {
				// 3rd deployment doesn't match
				currLabel := []map[string]string{doesNotMatch}
				if i == 0 || i == 1 {
					currLabel = []map[string]string{doesMatch}
				}

				deploymentListSrc.Items[i].Name = fmt.Sprintf("test%d", i)
				deploymentListSrc.Items[i].Spec.Applications[0].Targets = currLabel
			}

			resp, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Deployments)).To(Equal(2))
			Expect(resp.Deployments[0].Name).To(Equal("test0"))
			Expect(resp.Deployments[1].Name).To(Equal("test1"))
		})

		It("Filter: successfully returns 2 match label out of 3 with some labels not matching", func() {
			// populates a mock deployment object
			setDeploymentListObjects(&deploymentListSrc)

			deploymentListSrc.Items[0].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[1].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[2].Spec.DeploymentType = deploymentv1beta1.AutoScaling

			k8sClient := &nbmocks.FakeDeploymentV1{}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kClient := mockK8Client(ts.URL)

			Expect(err).ToNot(HaveOccurred())

			deploymentServer := NewDeployment(k8sClient, nil, kClient, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			var doesMatch = make(map[string]string)
			doesMatch["hello"] = "world"
			doesMatch["not"] = "valid"

			var doesNotMatch = make(map[string]string)
			doesNotMatch["foo"] = "world"

			for i := 0; i < 3; i++ {
				// 3rd deployment doesn't match
				currLabel := []map[string]string{doesNotMatch}
				if i == 0 || i == 1 {
					currLabel = []map[string]string{doesMatch}
				}

				deploymentListSrc.Items[i].Name = fmt.Sprintf("test%d", i)
				deploymentListSrc.Items[i].Spec.Applications[0].Targets = currLabel
			}

			resp, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.Deployments)).To(Equal(2))
			Expect(resp.Deployments[0].Name).To(Equal("test0"))
			Expect(resp.Deployments[1].Name).To(Equal("test1"))
		})

		It("Filter: fails due to invalid pattern whitespace in target label", func() {
			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "tester=foo ")

			_, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - labels[0]: value does not match regex " +
				"pattern `(^$)|^[a-z0-9]([-_.=,a-z0-9]{0,198}[a-z0-9])?$` [string.pattern]"))
		})

		It("Filter: fails due to invalid pattern non alphanumeric in target label", func() {
			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "tes?er=foo")

			_, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - labels[0]: value does not match regex " +
				"pattern `(^$)|^[a-z0-9]([-_.=,a-z0-9]{0,198}[a-z0-9])?$` [string.pattern]"))
		})

		It("Filter: fails due to invalid pattern uppercase char in target label", func() {
			matchingLabelList = matchingLabelList[:0]
			matchingLabelList = append(matchingLabelList, "tesTer=foo")

			_, err := deploymentServer.ListDeployments(context.Background(), &deploymentpb.ListDeploymentsRequest{
				Labels: matchingLabelList,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - labels[0]: value does not match regex " +
				"pattern `(^$)|^[a-z0-9]([-_.=,a-z0-9]{0,198}[a-z0-9])?$` [string.pattern]"))
		})
	})

	Describe("Gateway API Delete", func() {
		BeforeEach(func() {
			s.k8sClient = &nbmocks.FakeDeploymentV1{}

			// protovalidate Validator

			s.deploymentServer = NewDeployment(s.k8sClient, nil, nil, nil, nil, nil)

			// populates a mock deployment object
			setDeploymentListObject(&deploymentListSrc)

			deployInstance = SetDeployInstance(&deploymentListSrc, "")

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil)

			s.k8sClient.On(
				"Delete", context.Background(), "test-deployment",
				mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil)
		})

		It("successfully delete deployment", func() {
			defer ts.Close()

			k8sClient := &nbmocks.FakeDeploymentV1{}

			deploymentServer := NewDeployment(k8sClient, nil, s.kc, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"Delete", context.Background(), "test-deployment",
				mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil)

			_, err := deploymentServer.DeleteDeployment(context.Background(), &deploymentpb.DeleteDeploymentRequest{
				DeplId: VALID_UID,
			})

			Expect(err).ToNot(HaveOccurred())
		})

		It("fails to delete deployment due to invalid pattern uppercase char in deploy id", func() {
			_, err := s.deploymentServer.DeleteDeployment(context.Background(), &deploymentpb.DeleteDeploymentRequest{
				DeplId: "123456-123456-123456F",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("invalid DeleteDeploymentRequest.DeplId: value does not match regex " +
				"pattern \"^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$\""))
		})

		It("fails to delete deployment due to invalid pattern non alphanumeric in deploy id", func() {
			_, err := s.deploymentServer.DeleteDeployment(context.Background(), &deploymentpb.DeleteDeploymentRequest{
				DeplId: "123456-1234?56-123456",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - depl_id: value does not match regex " +
				"pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails to delete deployment due to invalid pattern whitespace in deploy id", func() {
			_, err := s.deploymentServer.DeleteDeployment(context.Background(), &deploymentpb.DeleteDeploymentRequest{
				DeplId: "123456-1234 56-123456",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - depl_id: value does not match regex " +
				"pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to missing depl-id in request body", func() {
			_, err := s.deploymentServer.DeleteDeployment(context.Background(), nil)

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("incomplete request"))
		})

		It("fails due to deployment not found", func() {
			_, err := s.deploymentServer.DeleteDeployment(context.Background(), &deploymentpb.DeleteDeploymentRequest{
				DeplId: INVALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal(fmt.Sprintf("deployment %v not found while deleting deployment", INVALID_UID)))
		})

		It("fails due to LIST error", func() {
			k8sClient := &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, errors.New("mock err"))

			k8sClient.On(
				"Delete", context.Background(), "test-deployment",
				mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil)

			_, err = deploymentServer.DeleteDeployment(context.Background(), &deploymentpb.DeleteDeploymentRequest{
				DeplId: VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock err"))
		})

		It("fails due to DELETE error", func() {
			k8sClient := &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil)

			k8sClient.On(
				"Delete", context.Background(), "test-deployment",
				mock.AnythingOfType("v1.DeleteOptions"),
			).Return(errors.New("mock DELETE err"))

			_, err := deploymentServer.DeleteDeployment(context.Background(), &deploymentpb.DeleteDeploymentRequest{
				DeplId: VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock DELETE err"))
		})

		It("successfully deletes deployment with error deleting secret", func() {
			defer ts.Close()

			k8sClient := &nbmocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, s.kc, nil, nil, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"Delete", context.Background(), "test-deployment",
				mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil)

			_, err := deploymentServer.DeleteDeployment(context.Background(), &deploymentpb.DeleteDeploymentRequest{
				DeplId: VALID_UID,
			})

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Gateway API Get", func() {
		BeforeEach(func() {
			// populates a mock deployment object
			setDeploymentListObject(&deploymentListSrc)

			setDeploymentClusterListObject(&deploymentClusterListSrc)

			deployInstance = SetDeployInstance(&deploymentListSrc, "")

			s.k8sClient = &nbmocks.FakeDeploymentV1{}

			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			s.kc = mockK8Client(ts.URL)

			// protovalidate Validator

			s.deploymentServer = NewDeployment(s.k8sClient, nil, s.kc, nil, nil, nil)

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"Get", context.Background(), mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

		})

		It("successfully get a deployment", func() {
			resp, err := s.deploymentServer.GetDeployment(context.Background(), &deploymentpb.GetDeploymentRequest{
				DeplId: VALID_UID,
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Deployment.Name).To(Equal("test-deployment"))
		})

		It("fails due to invalid pattern whitespace in deployment id", func() {
			_, err := s.deploymentServer.GetDeployment(context.Background(), &deploymentpb.GetDeploymentRequest{
				DeplId: "123456-1234 56-123456",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - depl_id: value does not match regex " +
				"pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to invalid pattern non alphanumeric in deployment id", func() {
			_, err := s.deploymentServer.GetDeployment(context.Background(), &deploymentpb.GetDeploymentRequest{
				DeplId: "123456-1234?56-123456",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - depl_id: value does not match regex " +
				"pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to invalid pattern uppercase char in deployment id", func() {
			_, err := s.deploymentServer.GetDeployment(context.Background(), &deploymentpb.GetDeploymentRequest{
				DeplId: "123456-1234F56-123456",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - depl_id: value does not match regex " +
				"pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to missing deployment id", func() {
			_, err := s.deploymentServer.GetDeployment(context.Background(), &deploymentpb.GetDeploymentRequest{
				DeplId: "",
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("incomplete request"))
		})

		It("fails due to deployment not found", func() {
			_, err := s.deploymentServer.GetDeployment(context.Background(), &deploymentpb.GetDeploymentRequest{
				DeplId: INVALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("deployment id 023456-023456-023456 not found"))
		})
	})

	Describe("Gateway API Create", func() {
		BeforeEach(func() {
			os.Setenv("USE_M2M_TOKEN", "true")
			os.Setenv("SECRET_SERVICE_ENABLED", "false")

			// populates a mock deployment object
			setDeploymentListObject(&deploymentListSrc)

			deploymentListSrc.Items[0].Spec.DeploymentType = deploymentv1beta1.AutoScaling
			deploymentListSrc.Items[0].Spec.DisplayName = "test display name 2"

			deployInstance = SetDeployInstance(&deploymentListSrc, "create")
			deployInstanceResp = getDeployInstance(&deploymentListSrc)

			s.k8sClient = &nbmocks.FakeDeploymentV1{}
			s.catalogClient = mockerymock.NewMockeryCatalogClient(GinkgoT())

			// protovalidate Validator

			// M2M auth client mock
			s.vaultAuthMock = &nbmocks.VaultAuth{}

			mockController := gomock.NewController(t)
			result := openpolicyagent.OpaResponse_Result{}
			_ = result.FromOpaResponseResult1(true)
			s.opaMock = openpolicyagent.NewMockClientWithResponsesInterface(mockController)
			s.opaMock.EXPECT().PostV1DataPackageRuleWithBodyWithResponse(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any()).Return(
				&openpolicyagent.PostV1DataPackageRuleResponse{
					JSON200: &openpolicyagent.OpaResponse{
						Result: result,
					},
				}, nil,
			).AnyTimes()

			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			s.kc = mockK8Client(ts.URL)

			s.deploymentServer = NewDeployment(s.k8sClient, s.opaMock, s.kc, nil, s.catalogClient, s.vaultAuthMock)

			s.k8sClient.On(
				"Create", nbmocks.AnyContextValue, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			md := metadata.Pairs("activeprojectid", VALID_PROJECT_ID, "authorization", "test-token")
			s.ctx = metadata.NewIncomingContext(context.Background(), md)
			s.vaultAuthMock.On("GetM2MToken", s.ctx).Return("test-m2m-token", nil).Once()
		})

		It("successfully create deployment with auto-scaling deployment type", func() {
			defer ts.Close()

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(string(deployInstance.Spec.DeploymentType)).Should(Equal("auto-scaling"))
			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("successfully create deployment with dp namespace name", func() {
			defer ts.Close()

			nbmocks.DpRespNsName.DeploymentPackage.Namespaces[0].Name = "test-test"

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("fails due to dp namespace name has prefix kind-", func() {
			defer ts.Close()

			nbmocks.DpRespNsName.DeploymentPackage.Namespaces[0].Name = "kind-test"

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespNsName, nil)

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("namespace name \"kind-test\" is invalid. Prefix \"kube-\", is reserved for Kubernetes system namespaces"))
		})

		It("fails due to dp namespace name missing", func() {
			defer ts.Close()

			nbmocks.DpRespNsName.DeploymentPackage.Namespaces[0].Name = ""

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespNsName, nil)

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing namespace name"))
		})

		It("fails due to dp namespace name is default", func() {
			defer ts.Close()

			nbmocks.DpRespNsName.DeploymentPackage.Namespaces[0].Name = "default"

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespNsName, nil)

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("namespace name \"default\" is invalid. Namespace name cannot be \"default\""))
		})

		It("fails due to dp namespace name does not conform with RFC 1123 label", func() {
			defer ts.Close()

			nbmocks.DpRespNsName.DeploymentPackage.Namespaces[0].Name = "TEST"

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespNsName, nil)

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("namespace name \"TEST\" is invalid. Use names that conforms with RFC 1123 label"))
		})

		It("fails due to parameter template mandatory is true but value missing", func() {
			defer ts.Close()

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPtResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("application wordpress is missing mandatory override profile values"))
		})

		It("fails due to parameter template mandatory is true and default value set", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Default = "test-default"

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPtResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("application : mandatory parameter template test-tp-value should have no default value"))
		})

		It("fails due to 2nd parameter template mandatory is true and default value set", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Default = ""

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPt2Resp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"global.admin_password\":\"foo\",\"global.another_password\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "test-appname",
				Values:  valuesStrPb,
			}

			nbmocks.AppHelmPt2Resp.Application.Profiles[0].ParameterTemplates[1].Default = "default-test"

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("application : mandatory parameter template test-tp2-value should have no default value"))
		})

		It("fails due to 2nd parameter template mandatory is true and value missing", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Default = ""

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPt2Resp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"global.admin_password\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "test-appname",
				Values:  valuesStrPb,
			}

			nbmocks.AppHelmPt2Resp.Application.Profiles[0].ParameterTemplates[1].Default = ""

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("application wordpress is missing mandatory override profile values"))
		})

		It("fails due to parameter template secret is true and default value set", func() {
			defer ts.Close()
			// deploymentServer = NewDeployment(k8sClient, nil, nil, nil, catalogClient, protoValidator, vaultAuthMock)

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Default = "kind-test"
			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Mandatory = false

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContextValue, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPtResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("application : secret parameter template test-tp-value should have no default value"))
		})

		It("successfully create deployment with parameter template mandatory is true and override value given", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Default = ""
			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Mandatory = true

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPtResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"global.admin_password\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "wordpress",
				Values:  valuesStrPb,
			}

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())

			s, _ := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
		})

		It("successfully create deployment with 2 parameter templates mandatory are true and both override value given", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Default = ""
			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Mandatory = true

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPt2Resp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"global.admin_password\":\"foo\",\"globals.another_password\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "wordpress",
				Values:  valuesStrPb,
			}

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())

			s, _ := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
		})

		It("successfully create deployment with parameter template mandatory is false and wrong override value given", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Default = ""
			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Mandatory = false

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPtResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"global.admin_password\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "test-appname",
				Values:  valuesStrPb,
			}

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())

			s, _ := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
		})

		It("successfully create deployment with 1 parameter template secret is true", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Default = ""
			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Mandatory = true
			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Secret = true

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPtResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"global.admin_password\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "wordpress",
				Values:  valuesStrPb,
			}

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())

			s, _ := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
		})

		It("successfully create deployment with 1 parameter template secret with 1level key", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Name = "single_password"

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPtResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"single_password\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "wordpress",
				Values:  valuesStrPb,
			}

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())

			s, _ := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
		})

		It("successfully create deployment with 1 parameter template secret with 7level key", func() {
			defer ts.Close()

			nbmocks.AppHelmPtResp.Application.Profiles[0].ParameterTemplates[0].Name = "one.two.three.four.five.six.seven"

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPtResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"one.two.three.four.five.six.seven\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "wordpress",
				Values:  valuesStrPb,
			}

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())

			s, _ := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
		})

		It("successfully create deployment with 2 parameter templates secrets to true", func() {
			defer ts.Close()

			nbmocks.AppHelmPt2Resp.Application.Profiles[0].ParameterTemplates[0].Default = ""
			nbmocks.AppHelmPt2Resp.Application.Profiles[0].ParameterTemplates[0].Mandatory = true
			nbmocks.AppHelmPt2Resp.Application.Profiles[0].ParameterTemplates[0].Secret = true
			nbmocks.AppHelmPt2Resp.Application.Profiles[0].ParameterTemplates[0].Name = "global.admin_password"
			nbmocks.AppHelmPt2Resp.Application.Profiles[0].ParameterTemplates[1].Mandatory = true
			nbmocks.AppHelmPt2Resp.Application.Profiles[0].ParameterTemplates[1].Secret = true

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmPt2Resp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			var valuesStrPb *structpb.Struct
			rawMsg := json.RawMessage("{\"global.admin_password\":\"foo\",\"globals.another_password\":\"foo\"}")
			_ = json.Unmarshal(rawMsg, &valuesStrPb)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "wordpress",
				Values:  valuesStrPb,
			}

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())

			s, _ := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
		})

		It("successfully create deployment with targeted deployment type", func() {
			defer ts.Close()

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			deployInstance.Spec.DeploymentType = deploymentv1beta1.Targeted

			Expect(string(deployInstance.Spec.DeploymentType)).Should(Equal("targeted"))
			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("successfully sets deployment type to auto-scaling when deploymentType is blank", func() {
			defer ts.Close()

			deployInstanceResp.DeploymentType = ""

			deployInstance = SetDeployInstance(&deploymentListSrc, "create")
			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			res, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(string(deployInstance.Spec.DeploymentType)).Should(Equal("auto-scaling"))
			Expect(res).NotTo(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails due to incomplete request", func() {
			deployInstanceResp = nil

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("incomplete request"))
		})

		It("fails due to missing targetClusters in request", func() {
			deployInstanceResp.TargetClusters = nil
			deployInstanceResp.AllAppTargetClusters = nil

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing targetClusters in request"))
		})

		It("fails due to request validation failed of deployment package app name", func() {
			deployInstanceResp.AppName = "ADM-TEST"

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - deployment.app_name: value " +
				"does not match regex pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to request validation failed of deployment package app version", func() {
			deployInstanceResp.AppVersion = "ADM TEST"

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - deployment.app_version: value " +
				"does not match regex pattern `^[a-z0-9][a-z0-9-.]{0,18}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to missing targetClusters.labels and targetClusters.clusterId", func() {
			var emptyLabels = make([]map[string]string, 0)
			deploymentListSrc.Items[0].Spec.Applications[0].Targets = emptyLabels

			s.k8sClient.On(
				"Create", nbmocks.AnyContext, deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName: "test-appname",
			}
			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing targetClusters.labels or targetClusters.clusterId in request"))
		})

		It("fails due to missing AllAppTargetClusters.labels and AllAppTargetClusters.clusterId", func() {
			deployInstanceResp.AllAppTargetClusters = &deploymentpb.TargetClusters{}
			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing allAppTargetClusters.labels or allAppTargetClusters.clusterId in request"))
		})

		It("fails due to missing allAppTargetClusters.labels when deployment type is auto-scaling", func() {
			deployInstanceResp.AllAppTargetClusters = &deploymentpb.TargetClusters{
				ClusterId: "test-ClusterId",
			}
			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("deployment type is auto-scaling but missing allAppTargetClusters.labels"))
		})

		It("fails due to missing allAppTargetClusters.clusterId when deployment type is targeted", func() {
			deployInstanceResp.DeploymentType = string(deploymentv1beta1.Targeted)

			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName:   "test-appname",
				ClusterId: "test-ClusterId",
			}

			deployInstanceResp.AllAppTargetClusters = &deploymentpb.TargetClusters{
				Labels: map[string]string{"test": "foo"},
			}
			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("deployment type is targeted but missing allAppTargetClusters.clusterId"))
		})

		It("fails due to missing targetClusters.labels when deployment type is auto-scaling", func() {
			var emptyLabels = make([]map[string]string, 0)
			deploymentListSrc.Items[0].Spec.Applications[0].Targets = emptyLabels

			s.k8sClient.On(
				"Create", context.Background(), deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName:   "test-appname",
				ClusterId: "test-ClusterId",
			}
			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("deployment type is auto-scaling but missing targetClusters.labels"))
		})

		It("fails due to missing targetClusters.clusterId when deployment type is targeted", func() {
			var emptyLabels = make([]map[string]string, 0)
			deploymentListSrc.Items[0].Spec.Applications[0].Targets = emptyLabels

			s.k8sClient.On(
				"Create", context.Background(), deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			deployInstanceResp.DeploymentType = string(deploymentv1beta1.Targeted)

			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName: "test-appname",
				Labels:  map[string]string{"test": "foo"},
			}
			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("deployment type is targeted but missing targetClusters.clusterId"))
		})

		It("fails due to missing targetClusters.AppName in request", func() {
			label := []map[string]string{{"test": "foo"}}
			deploymentListSrc.Items[0].Spec.Applications[0].Targets = label

			s.k8sClient.On(
				"Create", context.Background(), deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				Labels: map[string]string{"test": "foo"},
			}
			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing targetClusters.appName in request"))
		})

		It("fails due to missing OverrideValues.AppName in request", func() {
			s.k8sClient.On(
				"Create", context.Background(), deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{}

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - deployment.override_values[0].app_name: value " +
				"length must be at least 1 characters [string.min_len]\n - deployment.override_values[0].app_name: value " +
				"does not match regex pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to missing overrideValues.targetNamespace and overrideValues.values in request", func() {
			s.k8sClient.On(
				"Create", context.Background(), deployInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(deployInstance, nil)

			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "test-appname",
			}

			_, err := s.deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing overrideValues.targetNamespace or overrideValues.values in request"))
		})

		It("fails due to getting namespace", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Namespace", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()
			kc := mockK8Client(ts.URL)

			deploymentServer := NewDeployment(s.k8sClient, s.opaMock, kc, nil, s.catalogClient, s.vaultAuthMock)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetDockerRegReq).Return(&nbmocks.HelmRegResp, nil)

			_, err := deploymentServer.CreateDeployment(s.ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("the server does not allow this method on the requested resource (get namespaces 0000-1111-2222-3333-4444)"))
		})
	})

	Describe("Gateway API Update", func() {
		BeforeEach(func() {
			os.Setenv("SECRET_SERVICE_ENABLED", "false")

			// populates a mock deployment object
			setDeploymentListObject(&deploymentListSrc)

			deployInstance = SetDeployInstance(&deploymentListSrc, "")
			deployInstanceResp = getDeployInstance(&deploymentListSrc)

			s.k8sClient = &nbmocks.FakeDeploymentV1{}
			s.catalogClient = mockerymock.NewMockeryCatalogClient(GinkgoT())

			// protovalidate Validator

			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))
			s.kc = mockK8Client(ts.URL)

			// M2M auth client mock
			s.vaultAuthMock = &nbmocks.VaultAuth{}

			mockController := gomock.NewController(t)
			result := openpolicyagent.OpaResponse_Result{}
			_ = result.FromOpaResponseResult1(true)
			s.opaMock = openpolicyagent.NewMockClientWithResponsesInterface(mockController)
			s.opaMock.EXPECT().PostV1DataPackageRuleWithBodyWithResponse(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any()).Return(
				&openpolicyagent.PostV1DataPackageRuleResponse{
					JSON200: &openpolicyagent.OpaResponse{
						Result: result,
					},
				}, nil,
			).AnyTimes()

			s.deploymentServer = NewDeployment(s.k8sClient, s.opaMock, s.kc, nil, s.catalogClient, s.vaultAuthMock)

			s.k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"Update", nbmocks.AnyContext, mock.AnythingOfType("string"), deployInstance, mock.AnythingOfType("v1.UpdateOptions"),
			).Return(deployInstance, nil)

			md := metadata.Pairs("activeprojectid", VALID_PROJECT_ID, "authorization", "test-token")
			s.ctx = metadata.NewIncomingContext(context.Background(), md)
			s.vaultAuthMock.On("GetM2MToken", s.ctx).Return("test-m2m-token", nil).Once()

			// Mock the matchUIDDeployment function
			matchUIDDeploymentFn = func(ctx context.Context, uid, activeProjectID string, s *DeploymentSvc, opts metav1.ListOptions) (*deploymentv1beta1.Deployment, error) {
				// If we're using INVALID_UID, return an error
				if uid == INVALID_UID {
					return nil, fmt.Errorf("deployment %s not found", uid)
				}

				// Otherwise return a valid deployment
				return &deploymentv1beta1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-deployment",
						Namespace:       "test-namespace",
						ResourceVersion: "1",
					},
				}, nil
			}
		})

		AfterEach(func() {
			// Restore original function
			matchUIDDeploymentFn = origMatchUIDDeploymentFn
		})

		It("successfully update deployment", func() {
			defer ts.Close()

			s.k8sClient.On(
				"Update", nbmocks.AnyContext, mock.AnythingOfType("string"), deployInstance, mock.AnythingOfType("v1.UpdateOptions"),
			).Return(deployInstance, nil)

			s.k8sClient.On(
				"Delete", nbmocks.AnyContext, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil).Once()

			s.k8sClient.On(
				"Delete", nbmocks.AnyContextValue, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil).Once()

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			res, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(res).NotTo(BeNil())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("fails due to deployment not found", func() {
			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetDockerRegReq).Return(&nbmocks.DockerRegResp, nil)

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     INVALID_UID,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal(fmt.Sprintf("deployment %v not found while updating deployment", INVALID_UID)))
		})

		It("fails due to deployment resourceVersion not found", func() {
			matchUIDDeploymentFn = func(ctx context.Context, uid, activeProjectID string, s *DeploymentSvc, opts metav1.ListOptions) (*deploymentv1beta1.Deployment, error) {
				return &deploymentv1beta1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "test-namespace",
						// No ResourceVersion
					},
				}, nil
			}
			deploymentListSrc.Items[0].ObjectMeta.ResourceVersion = ""

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetDockerRegReq).Return(&nbmocks.DockerRegResp, nil)

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("resourceVersion not found while updating deployment"))
		})

		It("fails due to incomplete request", func() {
			deployInstanceResp = nil

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("incomplete request"))
		})

		It("fails due to missing targetClusters in request", func() {
			deployInstanceResp.TargetClusters = nil
			deployInstanceResp.AllAppTargetClusters = nil

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing targetClusters in request"))
		})

		It("fails due to request validation failed of deployment package app name", func() {
			deployInstanceResp.AppName = "ADM TEST"

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - deployment.app_name: value does not match " +
				"regex pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to request validation failed of deployment package app version", func() {
			deployInstanceResp.AppVersion = "ADM TEST"

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("validation error:\n - deployment.app_version: value does not " +
				"match regex pattern `^[a-z0-9][a-z0-9-.]{0,18}[a-z0-9]{0,1}$` [string.pattern]"))
		})

		It("fails due to missing targetClusters.labels and targetClusters.clusterId", func() {
			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName: "test-appname",
			}

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing targetClusters.labels or targetClusters.clusterId in request"))

		})

		It("fails due to missing targetClusters.labels when deployment type is auto-scaling", func() {
			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName:   "test-appname",
				ClusterId: "test-ClusterId",
			}

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("deployment type is auto-scaling but missing targetClusters.labels"))
		})

		It("fails due to missing targetClusters.clusterId when deployment type is targeted", func() {
			deployInstanceResp.DeploymentType = string(deploymentv1beta1.Targeted)

			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName: "test-appname",
				Labels:  map[string]string{"test": "foo"},
			}

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("deployment type is targeted but missing targetClusters.clusterId"))
		})

		It("fails due to missing targetClusters.AppName", func() {
			deployInstanceResp.TargetClusters[0] = &deploymentpb.TargetClusters{
				Labels: map[string]string{"test": "foo"},
			}

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing targetClusters.appName in request"))
		})

		It("fails due to missing overrideValues.targetNamespace and overrideValues.values in request", func() {
			deployInstanceResp.OverrideValues[0] = &deploymentpb.OverrideValues{
				AppName: "test-appname",
			}

			_, err := s.deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("missing overrideValues.targetNamespace or overrideValues.values in request"))
		})

		It("fails due deployment LIST error", func() {
			defer ts.Close()

			k8sClient = &nbmocks.FakeDeploymentV1{}

			matchUIDDeploymentFn = func(ctx context.Context, uid, activeProjectID string, s *DeploymentSvc, opts metav1.ListOptions) (*deploymentv1beta1.Deployment, error) {
				return nil, apierrors.NewInternalError(fmt.Errorf("internal error"))
			}

			deploymentServer = NewDeployment(k8sClient, nil, s.kc, nil, s.catalogClient, s.vaultAuthMock)
			deployInstance = SetDeployInstance(&deploymentListSrc, "")

			k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, errors.New("mock deployment list err")).Once()

			k8sClient.On(
				"Delete", nbmocks.AnyContext, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil).Once()

			k8sClient.On(
				"Delete", nbmocks.AnyContext, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil).Once()

			k8sClient.On(
				"Update", nbmocks.AnyContext, mock.AnythingOfType("string"), deployInstance, mock.AnythingOfType("v1.UpdateOptions"),
			).Return(deployInstance, nil)

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetDockerRegReq).Return(&nbmocks.DockerRegResp, nil)

			_, err := deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock deployment list err"))
		})

		It("fails due to UPDATE error", func() {
			defer ts.Close()

			k8sClient := &nbmocks.FakeDeploymentV1{}

			deploymentServer = NewDeployment(k8sClient, nil, s.kc, nil, s.catalogClient, s.vaultAuthMock)
			deployInstance = SetDeployInstance(&deploymentListSrc, "")

			k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"Update", nbmocks.AnyContext, mock.AnythingOfType("string"), deployInstance, mock.AnythingOfType("v1.UpdateOptions"),
			).Return(deployInstance, errors.New("mock update err"))

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppHelmResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)

			_, err := deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock update err"))
		})

		It("fails due to deleting secret", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))
			defer ts.Close()
			kc := mockK8Client(ts.URL)

			k8sClient := &nbmocks.FakeDeploymentV1{}

			deploymentServer = NewDeployment(k8sClient, nil, kc, nil, s.catalogClient, s.vaultAuthMock)
			deployInstance = SetDeployInstance(&deploymentListSrc, "")

			k8sClient.On(
				"List", nbmocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"Update", nbmocks.AnyContext, mock.AnythingOfType("string"), deployInstance, mock.AnythingOfType("v1.UpdateOptions"),
			).Return(deployInstance, nil)

			s.catalogClient.On("GetDeploymentPackage", nbmocks.AnyContext, nbmocks.AnyGetDpReq).Return(&nbmocks.DpRespGood, nil)
			s.catalogClient.On("GetApplication", nbmocks.AnyContext, nbmocks.AnyGetAppReq).Return(&nbmocks.AppResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetRegReq).Return(&nbmocks.HelmRegResp, nil)
			s.catalogClient.On("GetRegistry", nbmocks.AnyContext, nbmocks.AnyGetDockerRegReq).Return(&nbmocks.DockerRegResp, nil)

			_, err := deploymentServer.UpdateDeployment(s.ctx, &deploymentpb.UpdateDeploymentRequest{
				Deployment: deployInstanceResp,
				DeplId:     VALID_UID,
			})

			Expect(err).To(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("the server does not allow this method on the requested resource (delete secrets ProfileSecretName)"))
		})
	})

	Describe("Gateway API List Deployments Per Cluster", func() {
		BeforeEach(func() {
			s.k8sClient = &nbmocks.FakeDeploymentV1{}

			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			s.kc = mockK8Client(ts.URL)

			// protovalidate Validator

			mockController = gomock.NewController(t)
			s.opaMock = openpolicyagent.NewMockClientWithResponsesInterface(mockController)
			result := openpolicyagent.OpaResponse_Result{}
			_ = result.FromOpaResponseResult1(true)

			s.opaMock.EXPECT().PostV1DataPackageRuleWithBodyWithResponse(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any()).Return(
				&openpolicyagent.PostV1DataPackageRuleResponse{
					JSON200: &openpolicyagent.OpaResponse{
						Result: result,
					},
				}, nil,
			).AnyTimes()

			s.deploymentServer = NewDeployment(s.k8sClient, s.opaMock, s.kc, nil, nil, nil)

			// populates a mock deployment object
			setDeploymentListObject(&deploymentListSrc)
			setDeploymentClusterListObject(&deploymentClusterListSrc)

			s.k8sClient.On(
				"List", nbmocks.AnyContextValue, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentList{
				ListMeta: deploymentListSrc.ListMeta,
				TypeMeta: deploymentListSrc.TypeMeta,
				Items:    deploymentListSrc.Items,
			}, nil).Once()

			s.k8sClient.On(
				"List", nbmocks.AnyContextValue, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.DeploymentClusterList{
				ListMeta: deploymentClusterListSrc.ListMeta,
				TypeMeta: deploymentClusterListSrc.TypeMeta,
				Items:    deploymentClusterListSrc.Items,
			}, nil).Once()

			// clear list
			matchingLabelList = matchingLabelList[:0]
			md := metadata.Pairs("activeprojectid", VALID_PROJECT_ID)
			s.ctx = metadata.NewIncomingContext(context.Background(), md)
		})

		It("successfully returns a populated list with no filter labels", func() {
			defer ts.Close()

			resp, err := s.deploymentServer.ListDeploymentsPerCluster(s.ctx, &deploymentpb.ListDeploymentsPerClusterRequest{
				Labels:    matchingLabelList,
				ClusterId: VALID_CLUSTER_ID,
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.DeploymentInstancesCluster)).To(Equal(1))
		})
	})
})

func setDeployment() *Deployment {
	d := &Deployment{}

	d.Name = "test-name"
	d.AppName = "wordpress"
	d.AppVersion = "0.1.0"

	d.Namespace = VALID_PROJECT_ID
	if d.Namespace == "" {
		d.Namespace = d.Name
	}

	parameterTemplate := make([]catalogclient.ParameterTemplate, 1)

	helmAppList := []catalogclient.HelmApp{}
	helmApp := catalogclient.HelmApp{
		Name:               "test-name",
		Repo:               "test-repo",
		DefaultNamespace:   "test-deployment",
		ImageRegistry:      "test-imageRegistry",
		Chart:              "test-chart",
		Version:            "test-version",
		Profile:            "test-profile",
		Values:             "test-values",
		ParameterTemplates: parameterTemplate,
	}

	helmAppList = append(helmAppList, helmApp)

	d.HelmApps = &helmAppList

	OverrideValuesList := make([]*deploymentpb.OverrideValues, 1)
	var valuesStrPb *structpb.Struct

	rawMsg := json.RawMessage("{\"test\":\"foo\"}")
	_ = json.Unmarshal(rawMsg, &valuesStrPb)
	OverrideValuesList[0] = &deploymentpb.OverrideValues{
		AppName:         "test-appname",
		TargetNamespace: "apps",
		Values:          valuesStrPb,
	}

	d.OverrideValues = OverrideValuesList
	d.ParameterTemplateSecrets = make(map[string]string)

	return d
}

func setHelmApps(tt int) *Deployment {
	d := &Deployment{}

	parameterTemplate := make([]catalogclient.ParameterTemplate, 1)

	helmAppList := []catalogclient.HelmApp{}

	var helmApp catalogclient.HelmApp
	for i := range tt {
		helmApp = catalogclient.HelmApp{
			Name:               "test-name-" + strconv.Itoa(i),
			ParameterTemplates: parameterTemplate,
		}

		helmAppList = append(helmAppList, helmApp)
	}

	d.HelmApps = &helmAppList

	return d
}

func setDeploymentListObject(deploymentListSrc *deploymentv1beta1.DeploymentList) {
	deploymentListSrc.TypeMeta.Kind = KIND
	deploymentListSrc.TypeMeta.APIVersion = apiVersion

	deploymentListSrc.ListMeta.ResourceVersion = "6"
	deploymentListSrc.ListMeta.Continue = "yes"
	remainingItem := int64(10)
	deploymentListSrc.ListMeta.RemainingItemCount = &remainingItem

	deploymentListSrc.Items = make([]deploymentv1beta1.Deployment, 1)

	setDeploymentObject(&deploymentListSrc.Items[0])
}

func setDeploymentListObjects(deploymentListSrc *deploymentv1beta1.DeploymentList) {
	deploymentListSrc.TypeMeta.Kind = KIND
	deploymentListSrc.TypeMeta.APIVersion = apiVersion

	deploymentListSrc.ListMeta.ResourceVersion = "6"
	deploymentListSrc.ListMeta.Continue = "yes"
	remainingItem := int64(10)
	deploymentListSrc.ListMeta.RemainingItemCount = &remainingItem

	deploymentListSrc.Items = make([]deploymentv1beta1.Deployment, 3)

	setDeploymentObject(&deploymentListSrc.Items[0])
	setDeploymentObject(&deploymentListSrc.Items[1])
	setDeploymentObject(&deploymentListSrc.Items[2])
}

func setDeploymentClusterListObject(deploymentClusterListSrc *deploymentv1beta1.DeploymentClusterList) {
	deploymentClusterListSrc.TypeMeta.Kind = KIND_DC
	deploymentClusterListSrc.TypeMeta.APIVersion = apiVersion

	deploymentClusterListSrc.ListMeta.ResourceVersion = "6"
	deploymentClusterListSrc.ListMeta.Continue = "yes"
	remainingItem := int64(10)
	deploymentClusterListSrc.ListMeta.RemainingItemCount = &remainingItem

	deploymentClusterListSrc.Items = make([]deploymentv1beta1.DeploymentCluster, 3)
	setDeploymentClusterObject(&deploymentClusterListSrc.Items[0])
}

func setDeploymentClusterObject(deploymentClusterSrc *deploymentv1beta1.DeploymentCluster) {
	deploymentClusterSrc.ObjectMeta.Name = "test-deployment-cluster"
	deploymentClusterSrc.ObjectMeta.GenerateName = "test-generate-name"
	deploymentClusterSrc.ObjectMeta.Namespace = VALID_PROJECT_ID
	deploymentClusterSrc.ObjectMeta.UID = types.UID(VALID_UID_DC)
	deploymentClusterSrc.ObjectMeta.ResourceVersion = "6"
	deploymentClusterSrc.ObjectMeta.Generation = 24456
	deploymentClusterSrc.Spec.ClusterID = VALID_CLUSTERID
	deploymentClusterSrc.Spec.DeploymentID = VALID_UID

	currentTime := metav1.Now()
	deploymentClusterSrc.ObjectMeta.CreationTimestamp = currentTime
	deploymentClusterSrc.ObjectMeta.DeletionTimestamp = &currentTime
	deploymentClusterSrc.ObjectMeta.Labels = map[string]string{
		"app.kubernetes.io/name":                         "deployment-cluster",
		"app.kubernetes.io/instance":                     deploymentClusterSrc.ObjectMeta.Name,
		"app.kubernetes.io/part-of":                      "app-deployment-manager",
		"app.kubernetes.io/managed-by":                   "kustomize",
		"app.kubernetes.io/created-by":                   "app-deployment-manager",
		string(deploymentv1beta1.AppOrchActiveProjectID): VALID_PROJECT_ID,
		string(deploymentv1beta1.ClusterName):            VALID_CLUSTERID,
		string(deploymentv1beta1.DeploymentID):           VALID_UID,
	}

	deploymentClusterSrc.Status.Name = "test-cluster-displayname"
	status := deploymentv1beta1.Status{
		State:   "Running",
		Message: "",
		Summary: deploymentv1beta1.Summary{
			Total:   1,
			Running: 2,
			Down:    1,
			Unknown: 0,
		},
	}

	deploymentClusterSrc.Status.Apps = make([]deploymentv1beta1.App, 2)

	deploymentClusterSrc.Status.Apps[0] = deploymentv1beta1.App{
		Name:   "test0",
		Id:     "0.1.0",
		Status: status,
	}

	deploymentClusterSrc.Status.Apps[1] = deploymentv1beta1.App{
		Name:   "test1",
		Id:     "0.1.0",
		Status: status,
	}

	deploymentClusterSrc.Status.Status.State = "Running"
	deploymentClusterSrc.Status.Status.Message = ""
	deploymentClusterSrc.Status.Status.Summary.Total = 1
	deploymentClusterSrc.Status.Status.Summary.Running = 2
	deploymentClusterSrc.Status.Status.Summary.Down = 1
	deploymentClusterSrc.Status.Status.Summary.Unknown = 0
}

func setDeploymentObject(deploymentSrc *deploymentv1beta1.Deployment) {
	deploymentSrc.ObjectMeta.Name = "test-deployment"
	deploymentSrc.ObjectMeta.GenerateName = "test-generate-name"
	deploymentSrc.ObjectMeta.Namespace = VALID_PROJECT_ID
	deploymentSrc.ObjectMeta.UID = types.UID(VALID_UID)
	deploymentSrc.ObjectMeta.ResourceVersion = "6"
	deploymentSrc.ObjectMeta.Generation = 24456

	currentTime := metav1.Now()
	deploymentSrc.ObjectMeta.CreationTimestamp = currentTime
	deploymentSrc.ObjectMeta.DeletionTimestamp = &currentTime
	deploymentSrc.ObjectMeta.Labels = map[string]string{
		"app.kubernetes.io/name":                         "deployment",
		"app.kubernetes.io/instance":                     deploymentSrc.ObjectMeta.Name,
		"app.kubernetes.io/part-of":                      "app-deployment-manager",
		"app.kubernetes.io/managed-by":                   "kustomize",
		"app.kubernetes.io/created-by":                   "app-deployment-manager",
		string(deploymentv1beta1.AppOrchActiveProjectID): VALID_PROJECT_ID,
		string(deploymentv1beta1.ClusterName):            VALID_CLUSTERID,
	}

	deploymentSrc.Spec.DisplayName = "test display name 2"
	deploymentSrc.Spec.Project = "app.edge-orchestrator.intel.com"
	deploymentSrc.Spec.DeploymentPackageRef.Name = "wordpress"
	deploymentSrc.Spec.DeploymentPackageRef.Version = "0.1.0"
	deploymentSrc.Spec.DeploymentPackageRef.ProfileName = "default"
	deploymentSrc.Spec.DeploymentType = deploymentv1beta1.AutoScaling

	deploymentSrc.Spec.Applications = make([]deploymentv1beta1.Application, 1)
	deploymentSrc.Spec.Applications[0] = deploymentv1beta1.Application{
		Name:                "wordpress",
		NamespaceLabels:     map[string]string{},
		Namespace:           "test-deployment",
		ProfileSecretName:   "ProfileSecretName",
		ValueSecretName:     "ValueSecretName",
		DependsOn:           []string{"dependency"},
		EnableServiceExport: true,
		RedeployAfterUpdate: false,
		HelmApp: &deploymentv1beta1.HelmApp{
			Chart:   "wordpress",
			Version: "15.2.42",
			Repo:    "https://charts.bitnami.com/bitnami",
		},
	}

	var label []map[string]string
	deploymentSrc.Spec.Applications[0].Targets = label
	if deploymentSrc.Spec.DeploymentType == deploymentv1beta1.Targeted {
		deploymentSrc.Spec.Applications[0].Targets = []map[string]string{{"test-ClusterId-key": "test-ClusterId"}}
	} else {
		deploymentSrc.Spec.Applications[0].Targets = []map[string]string{{"test": "foo"}}
	}

	deploymentSrc.Status.Summary.Total = 3
	deploymentSrc.Status.Summary.Running = 2
	deploymentSrc.Status.Summary.Down = 1
	deploymentSrc.Status.Summary.Unknown = 0
}

func SetDeployInstance(deploymentListSrc *deploymentv1beta1.DeploymentList, scenario string) *deploymentv1beta1.Deployment {
	if scenario == "create" {
		app := make([]deploymentv1beta1.Application, 1)
		app[0] = deploymentv1beta1.Application{
			Name:                "wordpress",
			Version:             "0.1.0",
			Namespace:           "test-deployment",
			NamespaceLabels:     map[string]string{},
			Targets:             make([]map[string]string, 0),
			EnableServiceExport: true,
			DependsOn:           []string{"dependency"},
			RedeployAfterUpdate: false,
			HelmApp: &deploymentv1beta1.HelmApp{
				Chart:         "wordpress",
				Version:       "15.2.42",
				Repo:          "https://charts.bitnami.com/bitnami",
				ImageRegistry: "https://charts.bitnami.com/bitnami",
			},
			DependentDeploymentPackages: make(map[string]deploymentv1beta1.DeploymentPackageRef),
		}

		dpRef := deploymentv1beta1.DeploymentPackageRef{
			Name:        "wordpress",
			Version:     "0.1.0",
			ProfileName: "default",
		}

		instance := &deploymentv1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentListSrc.Items[0].ObjectMeta.Name,
				Namespace: deploymentListSrc.Items[0].ObjectMeta.Namespace,
				Labels:    deploymentListSrc.Items[0].ObjectMeta.Labels,
				UID:       types.UID(VALID_UID),
			},
			Spec: deploymentv1beta1.DeploymentSpec{
				DisplayName: deploymentListSrc.Items[0].Spec.DisplayName,
				Project:     deploymentListSrc.Items[0].Spec.Project,
				NetworkRef: corev1.ObjectReference{
					Name:       "network-1",
					Kind:       "Network",
					APIVersion: "network.edge-orchestrator.intel/v1",
				},
				DeploymentPackageRef: dpRef,
				Applications:         app,
				DeploymentType:       deploymentListSrc.Items[0].Spec.DeploymentType,
				ChildDeploymentList:  make(map[string]deploymentv1beta1.DependentDeploymentRef),
			},
		}
		return instance
	} else {
		app := make([]deploymentv1beta1.Application, 1)
		app[0] = deploymentv1beta1.Application{
			Name:                "wordpress",
			Version:             "0.1.0",
			Namespace:           "test-deployment",
			NamespaceLabels:     map[string]string{},
			Targets:             make([]map[string]string, 0),
			DependsOn:           []string{"dependency"},
			RedeployAfterUpdate: false,
			HelmApp: &deploymentv1beta1.HelmApp{
				Chart:         "wordpress",
				Version:       "15.2.42",
				Repo:          "https://charts.bitnami.com/bitnami",
				ImageRegistry: "https://charts.bitnami.com/bitnami",
			},
			DependentDeploymentPackages: make(map[string]deploymentv1beta1.DeploymentPackageRef),
		}

		dpRef := deploymentv1beta1.DeploymentPackageRef{
			Name:        "wordpress",
			Version:     "0.1.0",
			ProfileName: "default",
		}

		instance := &deploymentv1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            deploymentListSrc.Items[0].ObjectMeta.Name,
				Namespace:       deploymentListSrc.Items[0].ObjectMeta.Namespace,
				Labels:          deploymentListSrc.Items[0].ObjectMeta.Labels,
				ResourceVersion: deploymentListSrc.Items[0].ObjectMeta.ResourceVersion,
				UID:             types.UID(VALID_UID),
			},
			Spec: deploymentv1beta1.DeploymentSpec{
				DisplayName:          deploymentListSrc.Items[0].Spec.DisplayName,
				Project:              deploymentListSrc.Items[0].Spec.Project,
				DeploymentPackageRef: dpRef,
				Applications:         app,
				DeploymentType:       deploymentListSrc.Items[0].Spec.DeploymentType,
				NetworkRef:           deploymentListSrc.Items[0].Spec.NetworkRef,
			},
		}
		return instance
	}
}

func getClusterInstance() *deploymentpb.Cluster {
	// Create CR Deployment summary
	summary := &deploymentpb.Summary{
		Total:   1,
		Running: 1,
		Down:    0,
		Unknown: 0,
		Type:    "test-type",
	}

	// Create CR Deployment status
	status := &deploymentpb.Deployment_Status{
		State:   deploymentpb.State_RUNNING,
		Message: "test",
		Summary: summary,
	}

	appList := make([]*deploymentpb.App, 1)
	appList[0] = &deploymentpb.App{
		Name:   "test-app",
		Id:     "789465-789465",
		Status: status,
	}

	instance := &deploymentpb.Cluster{
		Name:   "test-cluster",
		Id:     "123456-123456-123465",
		Status: status,
		Apps:   appList,
	}

	return instance
}

func getDeployInstance(deploymentListSrc *deploymentv1beta1.DeploymentList) *deploymentpb.Deployment {
	OverrideValuesList := make([]*deploymentpb.OverrideValues, 1)
	var valuesStrPb *structpb.Struct

	rawMsg := json.RawMessage("{\"test\":\"foo\"}")
	_ = json.Unmarshal(rawMsg, &valuesStrPb)
	OverrideValuesList[0] = &deploymentpb.OverrideValues{
		AppName:         "test-appname",
		TargetNamespace: "apps",
		Values:          valuesStrPb,
	}

	TargetClustersList := make([]*deploymentpb.TargetClusters, 1)
	TargetClustersList[0] = &deploymentpb.TargetClusters{
		AppName: "test-appname",
		Labels:  map[string]string{"test": "foo"},
	}

	AllAppTargetClustersVal := &deploymentpb.TargetClusters{
		Labels: map[string]string{"test": "foobar"},
	}

	// Create CR Deployment summary
	summary := &deploymentpb.Summary{
		Total:   int32(deploymentListSrc.Items[0].Status.Summary.Total),
		Running: int32(deploymentListSrc.Items[0].Status.Summary.Running),
		Down:    int32(deploymentListSrc.Items[0].Status.Summary.Down),
		Unknown: int32(deploymentListSrc.Items[0].Status.Summary.Unknown),
		Type:    "test-type",
	}

	// Create CR Deployment status
	status := &deploymentpb.Deployment_Status{
		State:   deploymentpb.State_RUNNING,
		Message: "test",
		Summary: summary,
	}

	deployInstanceResp := &deploymentpb.Deployment{
		Name:                 deploymentListSrc.Items[0].ObjectMeta.Name,
		AppName:              deploymentListSrc.Items[0].Spec.DeploymentPackageRef.Name,
		DisplayName:          deploymentListSrc.Items[0].Spec.DisplayName,
		AppVersion:           deploymentListSrc.Items[0].Spec.DeploymentPackageRef.Version,
		ProfileName:          deploymentListSrc.Items[0].Spec.DeploymentPackageRef.ProfileName,
		NetworkName:          deploymentListSrc.Items[0].Spec.NetworkRef.Name,
		Status:               status,
		OverrideValues:       OverrideValuesList,
		TargetClusters:       TargetClustersList,
		AllAppTargetClusters: AllAppTargetClustersVal,
	}
	return deployInstanceResp
}

func mockK8Client(tsUrl string) *kubernetes.Clientset {
	config := &rest.Config{
		Host: tsUrl,
	}

	gv := metav1.SchemeGroupVersion
	config.GroupVersion = &gv

	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.ContentType = "application/json"

	_kClient, err := kubernetes.NewForConfig(config)
	Expect(err).ToNot(HaveOccurred())

	return _kClient
}
