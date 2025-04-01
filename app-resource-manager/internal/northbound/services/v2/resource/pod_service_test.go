// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package resource

import (
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
