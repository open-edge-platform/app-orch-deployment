// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	fuzz "github.com/AdaLogics/go-fuzz-headers"
	resourcev2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	sbmocks "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/southbound/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"
	"testing"
)

type FuzzTestSuite struct {
	ctx           context.Context
	cancel        context.CancelFunc
	sbHandlerMock *sbmocks.MockHandler
	server        Server
}

func setupFuzzTest(t *testing.T) *FuzzTestSuite {
	s := &FuzzTestSuite{}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.sbHandlerMock = sbmocks.NewMockHandler(t)
	s.server.sbHandler = s.sbHandlerMock

	return s

}

func FuzzGetVNC(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t)
	seedReq := &resourcev2.GetVNCRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	}
	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, seedData []byte) {

		consumer := fuzz.NewConsumer(seedData)
		req := &resourcev2.GetVNCRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}
		s.sbHandlerMock.On("AccessVMWithVNC", mock.AnythingOfType("*context.cancelCtx"),
			mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil, nil)

		resp, err := s.server.GetVNC(s.ctx, req)
		if err != nil {
			t.Log(err)
			assert.Nil(t, resp)
		}
	})
}

func FuzzStartVM(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t)
	seedReq := &resourcev2.StartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	}
	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &resourcev2.StartVirtualMachineRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}
		s.sbHandlerMock.On("StartVM", mock.AnythingOfType("*context.cancelCtx"),
			mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil)

		resp, err := s.server.StartVirtualMachine(s.ctx, req)
		if err != nil {
			t.Log(err)
			assert.Nil(t, resp)
		}

	})

}

func FuzzStopVM(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t)
	seedReq := &resourcev2.StopVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	}
	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &resourcev2.StopVirtualMachineRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.sbHandlerMock.On("StopVM", mock.AnythingOfType("*context.cancelCtx"),
			mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil)
		resp, err := s.server.StopVirtualMachine(s.ctx, req)
		if err != nil {
			assert.Nil(t, resp)
		}

	})

}

func FuzzRestartVM(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t)
	seedReq := &resourcev2.RestartVirtualMachineRequest{
		ClusterId:        testCluster1,
		AppId:            testApp1,
		VirtualMachineId: testVM1,
	}
	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &resourcev2.RestartVirtualMachineRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.sbHandlerMock.On("RestartVM", mock.AnythingOfType("*context.cancelCtx"),
			mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil)
		resp, err := s.server.RestartVirtualMachine(s.ctx, req)
		if err != nil {
			assert.Nil(t, resp)
		}
	})
}

func FuzzDeletePod(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t)
	seedReq := &resourcev2.DeletePodRequest{
		ClusterId: testCluster1,
		Namespace: testNamespace,
		PodName:   testPodName,
	}
	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(t, err)
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &resourcev2.DeletePodRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.sbHandlerMock.On("DeletePod", mock.AnythingOfType("*context.cancelCtx"),
			mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil)
		resp, err := s.server.DeletePod(s.ctx, req)
		if err != nil {
			assert.Nil(t, resp)
		}
	})
}

func FuzzListAppEndpoints(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t)
	seedReq := &resourcev2.ListAppEndpointsRequest{
		AppId:     testApp1,
		ClusterId: testCluster1,
	}
	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(t, err)
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &resourcev2.ListAppEndpointsRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.sbHandlerMock.On("GetAppEndpointsV2", mock.AnythingOfType("*context.cancelCtx"),
			mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil, nil)
		resp, err := s.server.ListAppEndpoints(s.ctx, req)
		if err != nil {
			assert.Nil(t, resp)
		}
	})
}

func FuzzListAppWorkloads(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t)
	seedReq := &resourcev2.ListAppWorkloadsRequest{
		AppId:     testApp1,
		ClusterId: testCluster1,
	}
	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(t, err)
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &resourcev2.ListAppWorkloadsRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.sbHandlerMock.On("GetAppWorkLoads", mock.AnythingOfType("*context.cancelCtx"),
			mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(nil, nil)
		resp, err := s.server.ListAppWorkloads(s.ctx, req)
		if err != nil {
			assert.Nil(t, resp)
		}
	})
}
