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

func (s *NorthboundTestSuite) TestDeletePod() {
	s.sbHandlerMock.On("DeletePod", mock.AnythingOfType("*context.valueCtx"),
		testCluster1, testNamespace, testPodName).Return(nil)

	_, err := s.podServiceClient.DeletePod(s.ctx, &resourceapiv2.DeletePodRequest{
		ClusterId: testCluster1,
		Namespace: testNamespace,
		PodName:   testPodName,
	})
	assert.NoError(s.T(), err)

}

func (s *NorthboundTestSuite) TestDeletePodError() {
	errMsg := "Failed to delete pod"
	s.sbHandlerMock.On("DeletePod", mock.AnythingOfType("*context.valueCtx"),
		testCluster1, testNamespace, testPodName).Return(errors.NewInternal(errMsg))

	_, err := s.podServiceClient.DeletePod(s.ctx, &resourceapiv2.DeletePodRequest{
		ClusterId: testCluster1,
		Namespace: testNamespace,
		PodName:   testPodName,
	})
	assert.Error(s.T(), err)
	assert.Equal(s.T(), errors.Status(errors.NewInternal(errMsg)).Err(), err)

}

func (s *NorthboundTestSuite) TestInvalidDeletePod() {
	_, err := s.podServiceClient.DeletePod(s.ctx, &resourceapiv2.DeletePodRequest{})
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))

}

func (s *NorthboundTestSuite) TestInvalidPodNameDeletePod() {
	resp, err := s.podServiceClient.DeletePod(s.ctx, &resourceapiv2.DeletePodRequest{
		ClusterId: testCluster1,
		Namespace: testNamespace,
		PodName:   testInvalidPodName,
	})
	assert.Nil(s.T(), resp)
	assert.Error(s.T(), err)

}

// TestDeletePod_MissingActiveProjectID tests the DeletePod API handler behavior when
// the request context is missing the ActiveProjectID.
func (s *NorthboundTestSuite) TestDeletePod_MissingActiveProjectID() {
	// Create context without ActiveProjectID
	ctxWithoutProjectID := context.Background()

	// Convert to a test client context
	testCtx := convertToTestContext(ctxWithoutProjectID)

	resp, err := s.podServiceClient.DeletePod(testCtx, &resourceapiv2.DeletePodRequest{
		ClusterId: testCluster1,
		Namespace: testNamespace,
		PodName:   testPodName,
	})

	assert.Error(s.T(), err)
	assert.Nil(s.T(), resp)
	assert.True(s.T(), errors.IsInvalid(errors.FromGRPC(err)))
	assert.Contains(s.T(), err.Error(), "activeprojectid")
}
