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
	appEndpointID1   = "15d47e26-61f0-4925-ada8-41200763af10"
	appEndpointName1 = "service-1"
	appEndpointID2   = "40a103c5-7bc4-410a-8398-adf54705f484"
	appEndpointName2 = "service-2"
)

func (s *NorthboundTestSuite) TestListAppEndpoints() {
	expectedOutput := []*resourceapiv2.AppEndpoint{
		{
			Id:   appEndpointID1,
			Name: appEndpointName1,
			Fqdns: []*resourceapiv2.Fqdn{
				{
					Fqdn: "example.org",
				},
			},
			Ports: []*resourceapiv2.Port{
				{
					Name:     "test-port-1",
					Protocol: "TCP",
					Value:    50000,
				},
			},
		},
		{
			Id:   appEndpointID2,
			Name: appEndpointName2,
			Fqdns: []*resourceapiv2.Fqdn{
				{
					Fqdn: "example.org",
				},
			},
			Ports: []*resourceapiv2.Port{
				{
					Name:     "test-port-2",
					Protocol: "TCP",
					Value:    50001,
				},
			},
		},
	}

	s.sbHandlerMock.On("GetAppEndpointsV2", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1).Return(expectedOutput, nil)

	resp, err := s.endpointServiceClient.ListAppEndpoints(s.ctx, &resourceapiv2.ListAppEndpointsRequest{
		ClusterId: testCluster1,
		AppId:     testApp1,
	})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedOutput[0].Id, resp.AppEndpoints[0].Id)
	assert.Equal(s.T(), expectedOutput[0].Name, resp.AppEndpoints[0].Name)
	assert.Equal(s.T(), expectedOutput[0].Fqdns[0].Fqdn, resp.AppEndpoints[0].Fqdns[0].Fqdn)
	assert.Equal(s.T(), expectedOutput[0].Ports[0].Name, resp.AppEndpoints[0].Ports[0].Name)

	assert.Equal(s.T(), expectedOutput[1].Id, resp.AppEndpoints[1].Id)
	assert.Equal(s.T(), expectedOutput[1].Name, resp.AppEndpoints[1].Name)
	assert.Equal(s.T(), expectedOutput[1].Fqdns[0].Fqdn, resp.AppEndpoints[1].Fqdns[0].Fqdn)
	assert.Equal(s.T(), expectedOutput[1].Ports[0].Name, resp.AppEndpoints[1].Ports[0].Name)

}

func (s *NorthboundTestSuite) TestInvalidAppIDListAppEndpoints() {
	resp, err := s.endpointServiceClient.ListAppEndpoints(s.ctx, &resourceapiv2.ListAppEndpointsRequest{
		AppId:     testInvalidAppID,
		ClusterId: testCluster1,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)

}

func (s *NorthboundTestSuite) TestInvalidListAppEndpoints() {
	_, err := s.endpointServiceClient.ListAppEndpoints(s.ctx, &resourceapiv2.ListAppEndpointsRequest{})
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))

}

func (s *NorthboundTestSuite) TestAppEndpointsError() {
	s.sbHandlerMock.On("GetAppEndpointsV2", mock.AnythingOfType("*context.valueCtx"),
		testApp1, testCluster1).Return(nil, errors.NewInternal("Failed to list application endpoints"))

	resp, err := s.endpointServiceClient.ListAppEndpoints(s.ctx, &resourceapiv2.ListAppEndpointsRequest{
		ClusterId: testCluster1,
		AppId:     testApp1,
	})
	assert.Nil(s.T(), resp)
	assert.Error(s.T(), err)

}

func (s *NorthboundTestSuite) TestListAppEndpoints_MissingActiveProjectID() {
	// Create context without ActiveProjectID
	ctxWithoutProjectID := context.Background()

	// Convert to a test client context
	testCtx := convertToTestContext(ctxWithoutProjectID)

	resp, err := s.endpointServiceClient.ListAppEndpoints(testCtx, &resourceapiv2.ListAppEndpointsRequest{
		ClusterId: testCluster1,
		AppId:     testApp1,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))
	assert.Contains(s.T(), err.Error(), "activeprojectid")
}
