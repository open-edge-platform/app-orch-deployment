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

// Test List Virtual Machines
func (s *NorthboundTestSuite) TestListVirtualMachineWorkloads() {
	expectedOutput := []*resourceapiv2.AppWorkload{
		{
			Id: testVM1,
		},
		{
			Id: testVM2,
		},
	}
	s.sbHandlerMock.On("GetAppWorkLoads", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1).Return(expectedOutput, nil)

	resp, err := s.appWorkloadServiceClient.ListAppWorkloads(s.ctx, &resourceapiv2.ListAppWorkloadsRequest{
		ClusterId: testCluster1,
		AppId:     testApp1,
	})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedOutput[0].Id, resp.AppWorkloads[0].Id)
	assert.Equal(s.T(), expectedOutput[1].Id, resp.AppWorkloads[1].Id)
}

func (s *NorthboundTestSuite) TestInvalidAppIDListVirtualMachineWorkLoads() {
	resp, err := s.appWorkloadServiceClient.ListAppWorkloads(s.ctx, &resourceapiv2.ListAppWorkloadsRequest{
		AppId:     testInvalidAppID,
		ClusterId: testCluster1,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)

}

func (s *NorthboundTestSuite) TestInvalidListVirtualMachineWorkloads() {
	_, err := s.appWorkloadServiceClient.ListAppWorkloads(s.ctx, &resourceapiv2.ListAppWorkloadsRequest{})
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))

}

func (s *NorthboundTestSuite) TestListVirtualMachineWorkloadsError() {
	s.sbHandlerMock.On("GetAppWorkLoads", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1).Return(nil, errors.NewInternal("test internal error"))
	resp, err := s.appWorkloadServiceClient.ListAppWorkloads(s.ctx, &resourceapiv2.ListAppWorkloadsRequest{
		ClusterId: testCluster1,
		AppId:     testApp1,
	})
	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)
}

func (s *NorthboundTestSuite) TestListAppWorkloads_MissingActiveProjectID() {
	// Create context without ActiveProjectID
	ctxWithoutProjectID := context.Background()

	// Convert to a test client context
	testCtx := convertToTestContext(ctxWithoutProjectID)

	resp, err := s.appWorkloadServiceClient.ListAppWorkloads(testCtx, &resourceapiv2.ListAppWorkloadsRequest{
		ClusterId: testCluster1,
		AppId:     testApp1,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))
	assert.Contains(s.T(), err.Error(), "activeprojectid")
}
