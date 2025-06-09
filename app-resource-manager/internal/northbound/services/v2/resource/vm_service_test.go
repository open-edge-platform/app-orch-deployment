// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"

	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testCluster1       = "test-cluster"
	testApp1           = "test-app1"
	testVM1            = "25533c09-d841-4e79-b558-df47bf59c2ea"
	testVM2            = "c99dbdb3-2c76-4239-bb76-ef9cd9171b0b"
	testVNCAddress     = "ws://test.vnc"
	testInvalidVMID    = "test-invalid-vm-id"
	testInvalidAppID   = "Test-Invalid-App-ID"
	testNamespace      = "test-namespace"
	testPodName        = "test-pod-name"
	testInvalidPodName = "InvalidPodName"
)

func (s *NorthboundTestSuite) TestInvalidGetVNC() {
	_, err := s.vmServiceClient.GetVNC(s.ctx, &resourceapiv2.GetVNCRequest{})
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))

}
func (s *NorthboundTestSuite) TestInvalidVMIDGetVNC() {
	resp, err := s.vmServiceClient.GetVNC(s.ctx, &resourceapiv2.GetVNCRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testInvalidVMID,
	})
	assert.Nil(s.T(), resp)
	assert.Error(s.T(), err)

}

func (s *NorthboundTestSuite) TestGetVNC() {
	s.sbHandlerMock.On("AccessVMWithVNC", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1, testVM1).Return(testVNCAddress, nil)

	resp, err := s.vmServiceClient.GetVNC(s.ctx, &resourceapiv2.GetVNCRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), testVNCAddress, resp.Address)

}

func (s *NorthboundTestSuite) TestGetVNCError() {
	s.sbHandlerMock.On("AccessVMWithVNC", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1, testVM1).Return("", errors.NewInternal("test internal error"))

	resp, err := s.vmServiceClient.GetVNC(s.ctx, &resourceapiv2.GetVNCRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})
	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)

}

// Test Stop Virtual Machine

func (s *NorthboundTestSuite) TestStopVirtualMachine() {
	s.sbHandlerMock.On("StopVM", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1, testVM1).Return(nil)

	_, err := s.vmServiceClient.StopVirtualMachine(s.ctx, &resourceapiv2.StopVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.NoError(s.T(), err)
}

func (s *NorthboundTestSuite) TestInvalidVMIDStopVirtualMachine() {
	resp, err := s.vmServiceClient.StopVirtualMachine(s.ctx, &resourceapiv2.StopVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testInvalidVMID,
	})
	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)

}

func (s *NorthboundTestSuite) TestInvalidStopVirtualMachine() {
	_, err := s.vmServiceClient.StopVirtualMachine(s.ctx, &resourceapiv2.StopVirtualMachineRequest{})
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))

}

func (s *NorthboundTestSuite) TestStopVirtualMachineError() {
	s.sbHandlerMock.On("StopVM", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1, testVM1).Return(errors.NewInternal("test internal error"))

	_, err := s.vmServiceClient.StopVirtualMachine(s.ctx, &resourceapiv2.StopVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.Error(s.T(), err)
}

// Test Start Virtual Machine

func (s *NorthboundTestSuite) TestStartVirtualMachine() {
	s.sbHandlerMock.On("StartVM", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1, testVM1).Return(nil)

	_, err := s.vmServiceClient.StartVirtualMachine(s.ctx, &resourceapiv2.StartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.NoError(s.T(), err)
}

func (s *NorthboundTestSuite) TestStartVirtualMachineError() {
	s.sbHandlerMock.On("StartVM", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1, testVM1).Return(errors.NewInternal("test internal error"))

	_, err := s.vmServiceClient.StartVirtualMachine(s.ctx, &resourceapiv2.StartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.Error(s.T(), err)
}

func (s *NorthboundTestSuite) TestInvalidVMIDStartVirtualMachine() {
	resp, err := s.vmServiceClient.StartVirtualMachine(s.ctx, &resourceapiv2.StartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testInvalidVMID,
	})
	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)

}

func (s *NorthboundTestSuite) TestInvalidStartVirtualMachine() {
	_, err := s.vmServiceClient.StartVirtualMachine(s.ctx, &resourceapiv2.StartVirtualMachineRequest{})
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))

}

// Test Restart Virtual Machine

func (s *NorthboundTestSuite) TestRestartVirtualMachine() {
	s.sbHandlerMock.On("RestartVM", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1, testVM1).Return(nil)

	_, err := s.vmServiceClient.RestartVirtualMachine(s.ctx, &resourceapiv2.RestartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.NoError(s.T(), err)
}

func (s *NorthboundTestSuite) TestInvalidVMIDRestartVirtualMachine() {
	resp, err := s.vmServiceClient.RestartVirtualMachine(s.ctx, &resourceapiv2.RestartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testInvalidVMID,
	})
	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)

}

func (s *NorthboundTestSuite) TestInvalidRestartVirtualMachine() {
	_, err := s.vmServiceClient.RestartVirtualMachine(s.ctx, &resourceapiv2.RestartVirtualMachineRequest{})
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))

}

func (s *NorthboundTestSuite) TestRestartVirtualMachineError() {
	s.sbHandlerMock.On("RestartVM", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1, testVM1).Return(errors.NewInternal("test internal error"))

	_, err := s.vmServiceClient.RestartVirtualMachine(s.ctx, &resourceapiv2.RestartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.Error(s.T(), err)
}

// Test MissingActiveProjectID for GetVNC, RestartVirtualMachine, StartVirtualMachine,
// StopVirtualMachine functions verifies that the methods returns an appropriate error
// when the request context does not contain an ActiveProjectID.
func (s *NorthboundTestSuite) TestGetVNC_MissingActiveProjectID() {
	// Create context without ActiveProjectID
	ctxWithoutProjectID := context.Background()

	// Convert to a test client context
	testCtx := convertToTestContext(ctxWithoutProjectID)

	resp, err := s.vmServiceClient.GetVNC(testCtx, &resourceapiv2.GetVNCRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)
	// Verify it's the right error type
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))
	// Verify error message
	assert.Contains(s.T(), err.Error(), "activeprojectid")
}

func (s *NorthboundTestSuite) TestStartVirtualMachine_MissingActiveProjectID() {
	// Create context without ActiveProjectID
	ctxWithoutProjectID := context.Background()

	// Convert to a test client context
	testCtx := convertToTestContext(ctxWithoutProjectID)

	resp, err := s.vmServiceClient.StartVirtualMachine(testCtx, &resourceapiv2.StartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))
	assert.Contains(s.T(), err.Error(), "activeprojectid")
}

func (s *NorthboundTestSuite) TestStopVirtualMachine_MissingActiveProjectID() {
	// Create context without ActiveProjectID
	ctxWithoutProjectID := context.Background()

	// Convert to a test client context
	testCtx := convertToTestContext(ctxWithoutProjectID)

	resp, err := s.vmServiceClient.StopVirtualMachine(testCtx, &resourceapiv2.StopVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))
	assert.Contains(s.T(), err.Error(), "activeprojectid")
}

func (s *NorthboundTestSuite) TestRestartVirtualMachine_MissingActiveProjectID() {
	// Create context without ActiveProjectID
	ctxWithoutProjectID := context.Background()

	// Convert to a test client context
	testCtx := convertToTestContext(ctxWithoutProjectID)

	resp, err := s.vmServiceClient.RestartVirtualMachine(testCtx, &resourceapiv2.RestartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))
	assert.Contains(s.T(), err.Error(), "activeprojectid")
}

func convertToTestContext(ctx context.Context) context.Context {
	return ctx
}
