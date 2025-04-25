// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"errors"

	"github.com/bufbuild/protovalidate-go"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"
	typederror "github.com/open-edge-platform/orch-library/go/pkg/errors"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
)

type TestReporter interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

var _ = Describe("Gateway gRPC Service", func() {
	var (
		ctx                context.Context
		deploymentListSrc  deploymentv1beta1.DeploymentList
		k8sClient          *mocks.FakeDeploymentV1
		protoValidator     *protovalidate.Validator
		deployInstanceResp *deploymentpb.Deployment
		mockController     *gomock.Controller
		deploymentServer   *DeploymentSvc
		t                  TestReporter
		err                error
	)

	Describe("Gateway API Authorization OPA", func() {
		BeforeEach(func() {
			k8sClient = &mocks.FakeDeploymentV1{}
			setDeploymentListObjects(&deploymentListSrc)
			deployInstanceResp = getDeployInstance(&deploymentListSrc)
			mockController = gomock.NewController(t)

			md := metadata.Pairs("foo", "bar")
			ctx = metadata.NewIncomingContext(context.Background(), md)

			// protovalidate Validator
			protoValidator, err = protovalidate.New()
			Expect(err).ToNot(HaveOccurred())
		})

		It("successfully authorizes", func() {
			opaMock := openpolicyagent.NewMockClientWithResponsesInterface(mockController)

			result := openpolicyagent.OpaResponse_Result{}
			err = result.FromOpaResponseResult1(true)

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

			deploymentServer = NewDeployment(k8sClient, opaMock, nil, nil, nil, protoValidator, nil)

			err = deploymentServer.AuthCheckAllowed(ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).ToNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("successfully returns denied access", func() {
			opaMock := openpolicyagent.NewMockClientWithResponsesInterface(mockController)

			result := openpolicyagent.OpaResponse_Result{}
			err = result.FromOpaResponseResult1(false)

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

			deploymentServer = NewDeployment(k8sClient, opaMock, nil, nil, nil, protoValidator, nil)

			err = deploymentServer.AuthCheckAllowed(ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			Expect(typederror.IsForbidden(err)).To(BeTrue())
			Expect(err.Error()).Should(Equal("access denied (3)"))
		})

		It("fails due to error on OPA Post", func() {
			opaMock := openpolicyagent.NewMockClientWithResponsesInterface(mockController)

			result := openpolicyagent.OpaResponse_Result{}
			err = result.FromOpaResponseResult1(true)

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
				}, errors.New("mock err"),
			).AnyTimes()

			deploymentServer = NewDeployment(k8sClient, opaMock, nil, nil, nil, protoValidator, nil)

			err = deploymentServer.AuthCheckAllowed(ctx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			Expect(typederror.IsInvalid(err)).To(BeTrue())
			Expect(err.Error()).Should(Equal("OPA rule CreateDeploymentRequest OPA Post error mock err"))
		})

		It("fails due to empty ctx and unable to extract info", func() {
			opaMock := openpolicyagent.NewMockClientWithResponsesInterface(mockController)

			result := openpolicyagent.OpaResponse_Result{}
			err = result.FromOpaResponseResult1(false)

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

			deploymentServer = NewDeployment(k8sClient, opaMock, nil, nil, nil, protoValidator, nil)

			emptyCtx := context.TODO()
			err = deploymentServer.AuthCheckAllowed(emptyCtx, &deploymentpb.CreateDeploymentRequest{
				Deployment: deployInstanceResp,
			})

			Expect(err).To(HaveOccurred())
			Expect(typederror.IsInvalid(err)).To(BeTrue())
			Expect(err.Error()).Should(Equal("unable to extract metadata from ctx"))
		})

	})

})
