// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubevirt

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	admmock "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm/mocks"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubevirt/mocks"
	countermock "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/wsproxy/mocks"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	v1 "kubevirt.io/api/core/v1"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	testAppID         = "appID1"
	wrongAppID        = "wrong"
	testNamespace     = ""
	testClusterID     = "cluster1"
	testVMID1         = "vm1"
	testVMID2         = "vm2"
	testKubeConfig    = `apiVersion: v1\nkind: Config\nclusters:\n- name: \"<CLUSTER_NAME>\"\n  cluster:\n    server: \"<URL>\"\n    certificate-authority-data: \"<TEST>\"\nusers:\n- name: \"<CLUSTER_NAME>\"\n  user:\n    token: \"<TEST>\"\ncontexts:\n- name: \"<CLUSTER_NAME>\"\n  context:\n    user: \"<CLUSTER_NAME>\"\n    cluster: \"<CLUSTER_NAME>\"\ncurrent-context: \"<CLUSTER_NAME>\"`
	expectedVNCAddr   = "wss://vnc.kind.internal/vnc/nil/appID1/cluster1/vm1"
	testOrigin        = "https://vnc.kind.internal"
	testForwardedHost = "vnc.kind.internal"
	testConfigValue   = `
 appDeploymentManager:
    endpoint: "http://adm-api.orch-app.svc:8081"
 webSocketServer:
    protocol: "wss"
    hostName: "vnc.kind.internal"
    sessionLimitPerIP: 0
    sessionLimitPerAccount: 0
    readLimitByte: 0
    dlIdleTimeoutMin: 0
    ulIdleTimeoutMin: 0
    allowedOrigins:
      - https://vnc.kind.internal`
)

var (
	testVirtualMachine1 = v1.VirtualMachine{
		ObjectMeta: k8sv1.ObjectMeta{
			UID:         testVMID1,
			Name:        testVMID1,
			Namespace:   testNamespace,
			Annotations: map[string]string{AnnotationKeyForAppID: testAppID, AnnotationKeyForVMDescription: "desc1"},
		},
		Spec: v1.VirtualMachineSpec{
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: k8sv1.ObjectMeta{
					UID:         testVMID1,
					Name:        testVMID2,
					Annotations: map[string]string{AnnotationKeyForAppID: testAppID, AnnotationKeyForVMDescription: "desc1"},
				},
			},
		},
	}
	testVirtualMachine2 = v1.VirtualMachine{
		ObjectMeta: k8sv1.ObjectMeta{
			UID:         testVMID1,
			Name:        testVMID2,
			Namespace:   testNamespace,
			Annotations: map[string]string{AnnotationKeyForAppID: testAppID, AnnotationKeyForVMDescription: "desc2"},
		},
		Spec: v1.VirtualMachineSpec{
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: k8sv1.ObjectMeta{
					UID:         testVMID2,
					Name:        testVMID2,
					Annotations: map[string]string{AnnotationKeyForAppID: testAppID, AnnotationKeyForVMDescription: "desc2"},
				},
			},
		},
	}
)

func openMockHTTPServer(f func(http.ResponseWriter, *http.Request)) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(f))
	return server
}

type KubevirtManagerTestSuite struct {
	suite.Suite
	mgr                                 Manager
	admMockClient                       *admmock.MockADMClient
	mockVirtualMachineInterface         *mocks.MockVirtualMachineInterface
	mockKubeVirtClient                  *mocks.MockKubevirtClient
	mockStreamInterface                 *mocks.MockStreamInterface
	mockVirtualMachineInstanceInterface *mocks.MockVirtualMachineInstanceInterface
	configFile                          *os.File
}

func (s *KubevirtManagerTestSuite) SetupSuite() {

}

func (s *KubevirtManagerTestSuite) TearDownSuite() {
}

func (s *KubevirtManagerTestSuite) SetupTest() {
	s.admMockClient = admmock.NewMockADMClient(s.T())
	s.mockKubeVirtClient = mocks.NewMockKubevirtClient(s.T())
	s.mockVirtualMachineInterface = mocks.NewMockVirtualMachineInterface(s.T())
	s.mockStreamInterface = mocks.NewMockStreamInterface(s.T())
	s.mockVirtualMachineInstanceInterface = mocks.NewMockVirtualMachineInstanceInterface(s.T())
	var err error
	err = os.Setenv("RATE_LIMITER_QPS", "25")
	assert.NoError(s.T(), err)
	err = os.Setenv("RATE_LIMITER_BURST", "1500")
	assert.NoError(s.T(), err)

	s.configFile, err = os.CreateTemp(s.T().TempDir(), "config.yaml")
	assert.NoError(s.T(), err)
	err = os.WriteFile(s.configFile.Name(), []byte(testConfigValue), 0600)
	assert.NoError(s.T(), err)

	s.mgr = NewManager(s.configFile.Name(), s.admMockClient, false)
	assert.NotNil(s.T(), s.mgr)

}

func (s *KubevirtManagerTestSuite) TearDownTest() {
}

func TestKubevirtManager(t *testing.T) {
	suite.Run(t, new(KubevirtManagerTestSuite))
}

func (s *KubevirtManagerTestSuite) TestManager_GetVMWorkloads() {
	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}
	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	vmWorkloads, err := s.mgr.GetVMWorkloads(ctx, testAppID, testClusterID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), vmWorkloads)
	assert.Equal(s.T(), 2, len(vmWorkloads))
}

func (s *KubevirtManagerTestSuite) TestManager_GetVMWorkloads_FailedGetKubevirtClient() {
	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(""), errors.NewUnknown("test"))

	vm, err := s.mgr.GetVMWorkloads(ctx, testAppID, testClusterID)
	assert.Nil(s.T(), vm)
	assert.Error(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_GetVMWorkloads_FailedGetKubevirtVirtualMachineList() {
	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(nil, errors.NewUnknown("test"))
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	vm, err := s.mgr.GetVMWorkloads(ctx, testAppID, testClusterID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), vm)
}

func (s *KubevirtManagerTestSuite) TestManager_StartVM() {
	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}

	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)
	s.mockVirtualMachineInterface.On("Start", ctx, testVMID1, &v1.StartOptions{}).Return(nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	err := s.mgr.StartVM(ctx, testAppID, testClusterID, testVMID1)
	assert.NoError(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_StartVM_FailedGetKubevirtClient() {
	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(""), errors.NewUnknown("test"))
	err := s.mgr.StartVM(ctx, testAppID, testClusterID, testVMID1)
	assert.Error(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_StartVM_FailedGetKubevirtVirtualMachine() {
	ctx := context.Background()

	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	err := s.mgr.StartVM(ctx, testAppID, testClusterID, "wrong")
	assert.Error(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_StopVM() {
	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}

	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)

	s.mockVirtualMachineInterface.On("Stop", ctx, testVMID1, &v1.StopOptions{}).Return(nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	err := s.mgr.StopVM(ctx, testAppID, testClusterID, testVMID1)
	assert.NoError(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_StopVM_FailedGetKubevirtClient() {
	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(""), errors.NewUnknown("test"))
	err := s.mgr.StopVM(ctx, testAppID, testClusterID, testVMID1)
	assert.Error(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_StopVM_FailedGetKubevirtVirtualMachine() {
	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}

	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	err := s.mgr.StopVM(ctx, testAppID, testClusterID, "wrong")
	assert.Error(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_RestartVM() {
	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}

	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)

	s.mockVirtualMachineInterface.On("Restart", ctx, testVMID1, &v1.RestartOptions{}).Return(nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	err := s.mgr.RestartVM(ctx, testAppID, testClusterID, testVMID1)
	assert.NoError(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_RestartVM_FailedGetKubevirtClient() {
	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(""), errors.NewUnknown("test"))
	err := s.mgr.RestartVM(ctx, testAppID, testClusterID, testVMID1)
	assert.Error(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_RestartVM_FailedGetKubevirtVirtualMachine() {
	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}

	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)

	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	err := s.mgr.RestartVM(ctx, testAppID, testClusterID, "wrong")
	assert.Error(s.T(), err)
}

func (s *KubevirtManagerTestSuite) TestManager_GetVNCAddress() {
	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}
	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)

	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	addr, err := s.mgr.GetVNCAddress(ctx, testAppID, testClusterID, testVMID1)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedVNCAddr, addr)
}

func (s *KubevirtManagerTestSuite) TestManager_GetVNCAddress_FailedGetConfigModel() {
	mgr := NewManager("", nil, false)
	assert.NotNil(s.T(), mgr)

	ctx := context.Background()

	addr, err := mgr.GetVNCAddress(ctx, testAppID, testClusterID, testVMID1)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), "", addr)
}

func (s *KubevirtManagerTestSuite) TestManager_GetVNCAddress_FailedGetKubevirtClient() {
	ctx := context.Background()
	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(""), errors.NewUnknown("test"))

	addr, err := s.mgr.GetVNCAddress(ctx, testAppID, testClusterID, testVMID1)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), "", addr)
}

func (s *KubevirtManagerTestSuite) TestManager_GetVNCAddress_FailedGetKubevirtVirtualMachine() {
	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}
	ctx := context.Background()
	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	addr, err := s.mgr.GetVNCAddress(ctx, testAppID, testClusterID, "wrong")
	assert.Error(s.T(), err)
	assert.Equal(s.T(), "", addr)
}

func (s *KubevirtManagerTestSuite) TestManager_GetVNCWebSocketHandler() {
	mockIPSessionCounter := countermock.NewMockCounter(s.T())
	mockAccountSessionCounter := countermock.NewMockCounter(s.T())
	ctx := context.Background()

	f := s.mgr.GetVNCWebSocketHandler(ctx, nil, mockIPSessionCounter, mockAccountSessionCounter)
	assert.NotNil(s.T(), f)
}

func (s *KubevirtManagerTestSuite) TestVNCWebSocketHandler() {

	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}

	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockStreamInterface.On("Stream", mock.Anything).Return(nil)
	s.mockVirtualMachineInstanceInterface.On("VNC", testVMID1).Return(s.mockStreamInterface, nil)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)
	s.mockKubeVirtClient.On("VirtualMachineInstance", "").Return(s.mockVirtualMachineInstanceInterface)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	mockIPSessionCounter := countermock.NewMockCounter(s.T())
	mockIPSessionCounter.On("Increase", mock.AnythingOfType("string")).Return(nil)
	mockIPSessionCounter.On("Decrease", mock.AnythingOfType("string")).Return(nil)
	mockIPSessionCounter.On("Print").Return("")
	mockAccountSessionCounter := countermock.NewMockCounter(s.T())
	mockAccountSessionCounter.On("Increase", mock.AnythingOfType("string")).Return(nil)
	mockAccountSessionCounter.On("Decrease", mock.AnythingOfType("string")).Return(nil)
	mockAccountSessionCounter.On("Print").Return("")

	f := s.mgr.GetVNCWebSocketHandler(ctx, nil, mockIPSessionCounter, mockAccountSessionCounter)
	server := openMockHTTPServer(f)
	defer server.Close()

	endpoint := fmt.Sprintf("ws://%s/vnc/%s/%s/%s/%s", server.URL[7:], "test1", testAppID, testClusterID, testVMID1)
	req, err := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("X-Forwarded-Host", testForwardedHost)
	assert.NoError(s.T(), err)
	conn, resp, err := websocket.DefaultDialer.Dial(endpoint, req.Header)

	assert.NotNil(s.T(), conn)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusSwitchingProtocols, resp.StatusCode)
	s.T().Cleanup(func() {
		assert.NoError(s.T(), mockIPSessionCounter.Increase(""))
		assert.NoError(s.T(), mockIPSessionCounter.Decrease(""))
		assert.Equal(s.T(), mockIPSessionCounter.Print(), "")
		assert.NoError(s.T(), mockAccountSessionCounter.Increase(""))
		assert.NoError(s.T(), mockAccountSessionCounter.Decrease(""))
		assert.Equal(s.T(), mockAccountSessionCounter.Print(), "")
	})
}

func (s *KubevirtManagerTestSuite) TestVNCWebSocketHandler_FailedGetKubevirtClient() {
	ctx := context.Background()

	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(""), errors.NewUnknown("test"))

	mockIPSessionCounter := countermock.NewMockCounter(s.T())
	mockIPSessionCounter.On("Increase", mock.AnythingOfType("string")).Return(nil)
	mockIPSessionCounter.On("Decrease", mock.AnythingOfType("string")).Return(nil)
	mockIPSessionCounter.On("Print").Return("")
	mockAccountSessionCounter := countermock.NewMockCounter(s.T())
	mockAccountSessionCounter.On("Increase", mock.AnythingOfType("string")).Return(nil)
	mockAccountSessionCounter.On("Decrease", mock.AnythingOfType("string")).Return(nil)
	mockAccountSessionCounter.On("Print").Return("")

	f := s.mgr.GetVNCWebSocketHandler(ctx, nil, mockIPSessionCounter, mockAccountSessionCounter)
	server := openMockHTTPServer(f)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/vnc/nil/%s/%s/%s", server.URL, testAppID, testClusterID, testVMID1), nil)
	assert.NoError(s.T(), err)
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("X-Forwarded-Host", testForwardedHost)
	resp, err := client.Do(req)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
	s.T().Cleanup(func() {
		assert.NoError(s.T(), mockIPSessionCounter.Increase(""))
		assert.NoError(s.T(), mockIPSessionCounter.Decrease(""))
		assert.Equal(s.T(), mockIPSessionCounter.Print(), "")
		assert.NoError(s.T(), mockAccountSessionCounter.Increase(""))
		assert.NoError(s.T(), mockAccountSessionCounter.Decrease(""))
		assert.Equal(s.T(), mockAccountSessionCounter.Print(), "")
	})
}

func (s *KubevirtManagerTestSuite) TestVNCWebSocketHandler_FailedToGetVNCStream_FailedToGetVM() {

	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}
	ctx := context.Background()
	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)
	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	mockIPSessionCounter := countermock.NewMockCounter(s.T())
	mockIPSessionCounter.On("Increase", mock.AnythingOfType("string")).Return(nil)
	mockIPSessionCounter.On("Decrease", mock.AnythingOfType("string")).Return(nil)
	mockIPSessionCounter.On("Print").Return("")
	mockAccountSessionCounter := countermock.NewMockCounter(s.T())
	mockAccountSessionCounter.On("Increase", mock.AnythingOfType("string")).Return(nil)
	mockAccountSessionCounter.On("Decrease", mock.AnythingOfType("string")).Return(nil)
	mockAccountSessionCounter.On("Print").Return("")

	f := s.mgr.GetVNCWebSocketHandler(ctx, nil, mockIPSessionCounter, mockAccountSessionCounter)
	server := openMockHTTPServer(f)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/vnc/nil/%s/%s/%s", server.URL, testAppID, testClusterID, "wrong"), nil)
	assert.NoError(s.T(), err)
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("X-Forwarded-Host", testForwardedHost)
	resp, err := client.Do(req)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
	s.T().Cleanup(func() {
		assert.NoError(s.T(), mockIPSessionCounter.Increase(""))
		assert.NoError(s.T(), mockIPSessionCounter.Decrease(""))
		assert.Equal(s.T(), mockIPSessionCounter.Print(), "")
		assert.NoError(s.T(), mockAccountSessionCounter.Increase(""))
		assert.NoError(s.T(), mockAccountSessionCounter.Decrease(""))
		assert.Equal(s.T(), mockAccountSessionCounter.Print(), "")
	})
}

func (s *KubevirtManagerTestSuite) TestVNCWebSocketHandler_FailedCallUpgrade() {

	vmList := v1.VirtualMachineList{
		Items: []v1.VirtualMachine{testVirtualMachine1, testVirtualMachine2},
	}
	ctx := context.Background()
	s.admMockClient.On("GetKubeConfig", ctx, testClusterID).Return([]byte(testKubeConfig), nil)

	s.mockVirtualMachineInterface.On("List", ctx, &k8sv1.ListOptions{}).Return(&vmList, nil)
	s.mockVirtualMachineInstanceInterface.On("VNC", testVMID1).Return(s.mockStreamInterface, nil)
	s.admMockClient.On("GetAppNamespace", ctx, testAppID).Return(testNamespace, nil)
	s.mockKubeVirtClient.On("VirtualMachineInstance", "").Return(s.mockVirtualMachineInstanceInterface)
	s.mockKubeVirtClient.On("VirtualMachine", testNamespace).Return(s.mockVirtualMachineInterface)

	origK8sRESTConfigFromKubeConfig := k8sRESTConfigFromKubeConfig
	defer func() { k8sRESTConfigFromKubeConfig = origK8sRESTConfigFromKubeConfig }()
	k8sRESTConfigFromKubeConfig = getMockK8sRESTConfigFromKubeConfig(&rest.Config{}, nil)

	origKubeVirtGetKubevirtClientFromRESTConfig := kubeVirtGetKubevirtClientFromRESTConfig
	defer func() { kubeVirtGetKubevirtClientFromRESTConfig = origKubeVirtGetKubevirtClientFromRESTConfig }()
	kubeVirtGetKubevirtClientFromRESTConfig = getMockKubeVirtGetKubevirtClientFromRESTConfig(s.mockKubeVirtClient, nil)

	mockIPSessionCounter := countermock.NewMockCounter(s.T())
	mockIPSessionCounter.On("Increase", mock.AnythingOfType("string")).Return(nil)
	mockIPSessionCounter.On("Decrease", mock.AnythingOfType("string")).Return(nil)
	mockIPSessionCounter.On("Print").Return("")
	mockAccountSessionCounter := countermock.NewMockCounter(s.T())
	mockAccountSessionCounter.On("Increase", mock.AnythingOfType("string")).Return(nil)
	mockAccountSessionCounter.On("Decrease", mock.AnythingOfType("string")).Return(nil)
	mockAccountSessionCounter.On("Print").Return("")

	f := s.mgr.GetVNCWebSocketHandler(ctx, nil, mockIPSessionCounter, mockAccountSessionCounter)
	server := openMockHTTPServer(f)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/vnc/%s/%s/%s/%s", server.URL, "nil", testAppID, testClusterID, testVMID1), nil)
	assert.NoError(s.T(), err)
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("X-Forwarded-Host", testForwardedHost)
	resp, err := client.Do(req)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
	s.T().Cleanup(func() {
		assert.NoError(s.T(), mockIPSessionCounter.Increase(""))
		assert.NoError(s.T(), mockIPSessionCounter.Decrease(""))
		assert.Equal(s.T(), mockIPSessionCounter.Print(), "")
		assert.NoError(s.T(), mockAccountSessionCounter.Increase(""))
		assert.NoError(s.T(), mockAccountSessionCounter.Decrease(""))
		assert.Equal(s.T(), mockAccountSessionCounter.Print(), "")
	})
}

func (s *KubevirtManagerTestSuite) TestVNCWebSocketHandler_WrongOrigin() {
	mockIPSessionCounter := countermock.NewMockCounter(s.T())
	mockAccountSessionCounter := countermock.NewMockCounter(s.T())
	ctx := context.Background()

	f := s.mgr.GetVNCWebSocketHandler(ctx, nil, mockIPSessionCounter, mockAccountSessionCounter)
	server := openMockHTTPServer(f)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/vnc/nil/%s/%s/%s", server.URL, testAppID, testClusterID, testVMID1), nil)
	assert.NoError(s.T(), err)
	req.Header.Set("Origin", "wrong")
	req.Header.Set("X-Forwarded-Host", testForwardedHost)
	resp, err := client.Do(req)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusForbidden, resp.StatusCode)
}

func (s *KubevirtManagerTestSuite) TestVNCWebSocketHandler_WrongForwardedHost() {
	ctx := context.Background()

	mockIPSessionCounter := countermock.NewMockCounter(s.T())
	mockAccountSessionCounter := countermock.NewMockCounter(s.T())
	f := s.mgr.GetVNCWebSocketHandler(ctx, nil, mockIPSessionCounter, mockAccountSessionCounter)
	server := openMockHTTPServer(f)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/vnc/nil/%s/%s/%s", server.URL, testAppID, testClusterID, testVMID1), nil)
	assert.NoError(s.T(), err)
	req.Header.Set("Origin", "wrong")
	req.Header.Set("X-Forwarded-Host", testForwardedHost)
	resp, err := client.Do(req)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusForbidden, resp.StatusCode)
}
