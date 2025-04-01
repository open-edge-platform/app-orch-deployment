// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	sbmocks "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/southbound/mocks"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
	"time"
)

type NorthboundTestSuite struct {
	suite.Suite
	ctx                      context.Context
	cancel                   context.CancelFunc
	conn                     *grpc.ClientConn
	startTime                time.Time
	sbHandlerMock            *sbmocks.MockHandler
	vmServiceClient          resourceapiv2.VirtualMachineServiceClient
	endpointServiceClient    resourceapiv2.EndpointsServiceClient
	appWorkloadServiceClient resourceapiv2.AppWorkloadServiceClient
	podServiceClient         resourceapiv2.PodServiceClient
	opa                      openpolicyagent.ClientWithResponsesInterface
}

func (s *NorthboundTestSuite) SetupSuite() {

}

func (s *NorthboundTestSuite) TearDownSuite() {
}

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func newTestService(_ *testing.T, mockHandler *sbmocks.MockHandler, opaClient openpolicyagent.ClientWithResponsesInterface) Service {
	return Service{
		sbHandler: mockHandler,
		opaClient: opaClient,
	}
}

func createServer(t *testing.T, sbMockHandler *sbmocks.MockHandler, opaClient openpolicyagent.ClientWithResponsesInterface) error {
	lis = bufconn.Listen(1024 * 1024)
	s := newTestService(t, sbMockHandler, opaClient)
	s.sbHandler = sbMockHandler
	s.opaClient = opaClient

	assert.NotNil(t, s)
	server := grpc.NewServer()
	s.Register(server)

	go func() {
		if err := server.Serve(lis); err != nil {
			assert.NoError(t, err, "Server exited with error: %v", err)
		}
	}()
	return nil
}

func newConnection() (*grpc.ClientConn, error) {
	ctx := context.Background()
	// nolint:staticcheck
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	return conn, err
}

func (s *NorthboundTestSuite) SetupTest() {
	s.T().Log("Setting up the test suite")
	s.sbHandlerMock = sbmocks.NewMockHandler(s.T())
	mockController := gomock.NewController(s.T())
	opaMock := openpolicyagent.NewMockClientWithResponsesInterface(mockController)
	result := openpolicyagent.OpaResponse_Result{}
	err := result.FromOpaResponseResult1(true)
	s.NoError(err)
	opaMock.EXPECT().PostV1DataPackageRuleWithBodyWithResponse(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		&openpolicyagent.PostV1DataPackageRuleResponse{
			JSON200: &openpolicyagent.OpaResponse{
				DecisionId: nil,
				Metrics:    nil,
				Result:     result,
			},
		}, nil,
	).AnyTimes()

	s.opa = opaMock

	err = createServer(s.T(), s.sbHandlerMock, s.opa)
	assert.NoError(s.T(), err)
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.conn, err = newConnection()
	s.startTime = time.Now()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), s.conn)
	s.vmServiceClient = resourceapiv2.NewVirtualMachineServiceClient(s.conn)
	s.endpointServiceClient = resourceapiv2.NewEndpointsServiceClient(s.conn)
	s.appWorkloadServiceClient = resourceapiv2.NewAppWorkloadServiceClient(s.conn)
	s.podServiceClient = resourceapiv2.NewPodServiceClient(s.conn)
	assert.NotNil(s.T(), s.vmServiceClient)
}

func (s *NorthboundTestSuite) TearDownTest() {
	if s.conn != nil {
		_ = s.conn.Close()
		s.cancel()
	}
	s.conn = nil
}

func TestNewService(t *testing.T) {
	s := NewService(sbmocks.NewMockHandler(t), nil)
	assert.NotNil(t, s)
}

func TestNorthbound(t *testing.T) {
	suite.Run(t, new(NorthboundTestSuite))
}
