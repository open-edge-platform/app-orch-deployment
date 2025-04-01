// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	resourcev2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	admmock "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm/mocks"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubernetes"
	k8smocks "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubernetes/mocks"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubevirt"
	kvmocks "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubevirt/mocks"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/stretchr/testify/assert"

	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testConfigPath = ""
	testAppID1     = "appID1"
	testCluster1   = "cluster1"
	testPodName    = "testPod1"
	testNamespace  = "testNamespace"
	testVM1        = "vm1"
	testKubeConfig = `apiVersion: v1\nkind: Config\nclusters:\n- name: \"<CLUSTER_NAME>\"\n  cluster:\n    server: \"<URL>\"\n    certificate-authority-data: \"<TEST>\"\nusers:\n- name: \"<CLUSTER_NAME>\"\n  user:\n    token: \"<TEST>\"\ncontexts:\n- name: \"<CLUSTER_NAME>\"\n  context:\n    user: \"<CLUSTER_NAME>\"\n    cluster: \"<CLUSTER_NAME>\"\ncurrent-context: \"<CLUSTER_NAME>\"`
)

func TestNewHandler(t *testing.T) {
	t.Skip()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	admMockClient := admmock.NewMockADMClient(t)
	admMockClient.On("GetKubeConfig", context.Background(), testCluster1).Return([]byte(testKubeConfig), nil)

	k8s := kubernetes.NewManager("", admMockClient)
	kv := kubevirt.NewManager("", admMockClient, false)
	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

}

func TestHandler_StartVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	ctx := context.Background()

	kv.On("StartVM", ctx, testAppID1, testCluster1, testVM1).Return(nil)

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	err := h.StartVM(ctx, testAppID1, testCluster1, testVM1)
	assert.NoError(t, err)
}

func TestHandler_StopVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	ctx := context.Background()

	kv.On("StopVM", ctx, testAppID1, testCluster1, testVM1).Return(nil)

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	err := h.StopVM(ctx, testAppID1, testCluster1, testVM1)
	assert.NoError(t, err)
}

func TestHandler_RestartVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	ctx := context.Background()

	kv.On("RestartVM", ctx, testAppID1, testCluster1, testVM1).Return(nil)

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	err := h.RestartVM(ctx, testAppID1, testCluster1, testVM1)
	assert.NoError(t, err)
}

func TestHandler_AccessVMWithVNC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	ctx := context.Background()

	kv.On("GetVNCAddress", ctx, testAppID1, testCluster1, testVM1).Return("address", nil)

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	addr, err := h.AccessVMWithVNC(ctx, testAppID1, testCluster1, testVM1)
	assert.NoError(t, err)
	assert.Equal(t, "address", addr)
}

func TestHandler_GetAppEndpointsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	expectedOutput := []*resourcev2.AppEndpoint{
		{
			Id:   "078c36c6-e964-4dcd-bdde-27f0df591383",
			Name: "service-1",
			Fqdns: []*resourcev2.Fqdn{
				{
					Fqdn: "example.org",
				},
			},
			Ports: []*resourcev2.Port{
				{
					Name:     "test-port-1",
					Value:    50000,
					Protocol: "TCP",
				},
			},
		},
	}
	ctx := context.Background()

	k8s.On("GetAppEndpointsV2", ctx, testAppID1, testCluster1).Return(expectedOutput, nil)

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	endpoints, err := h.GetAppEndpointsV2(ctx, testAppID1, testCluster1)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput[0].Id, endpoints[0].Id)
	assert.Equal(t, expectedOutput[0].Name, endpoints[0].Name)
	assert.Equal(t, expectedOutput[0].Fqdns[0].Fqdn, endpoints[0].Fqdns[0].Fqdn)

}

func TestHandler_GetAppWorkloads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	ctx := context.Background()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	k8s.On("GetPodWorkloads", ctx, testAppID1, testCluster1).Return([]*resourcev2.AppWorkload{}, nil)
	kv.On("GetVMWorkloads", ctx, testAppID1, testCluster1).Return([]*resourcev2.AppWorkload{}, nil)

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	workloads, err := h.GetAppWorkLoads(ctx, testAppID1, testCluster1)
	assert.NotNil(t, workloads)
	assert.NoError(t, err)
}

func TestHandler_GetAppWorkloadsVMWorkloadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	ctx := context.Background()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	kv.On("GetVMWorkloads", ctx, testAppID1, testCluster1).Return(nil, errors.NewInternal("internal error"))

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	workloads, err := h.GetAppWorkLoads(ctx, testAppID1, testCluster1)
	assert.Nil(t, workloads)
	assert.True(t, errors.IsInternal(err))
}

func TestHandler_GetAppWorkloadsPodWorkloadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	ctx := context.Background()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	kv.On("GetVMWorkloads", ctx, testAppID1, testCluster1).Return([]*resourcev2.AppWorkload{}, nil)
	k8s.On("GetPodWorkloads", ctx, testAppID1, testCluster1).Return(nil, errors.NewInternal("internal error"))

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	workloads, err := h.GetAppWorkLoads(ctx, testAppID1, testCluster1)
	assert.Nil(t, workloads)
	assert.True(t, errors.IsInternal(err))
}

func TestHandler_GetAppWorkloadsWithItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	ctx := context.Background()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)
	k8s.On("GetPodWorkloads", ctx, testAppID1, testCluster1).Return([]*resourcev2.AppWorkload{
		{
			Id: "test-pod-workload-1",
		},
	}, nil)
	kv.On("GetVMWorkloads", ctx, testAppID1, testCluster1).Return([]*resourcev2.AppWorkload{
		{
			Id: "test-vm-workload-1",
		},
	}, nil)

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)

	workloads, err := h.GetAppWorkLoads(ctx, testAppID1, testCluster1)
	assert.NotNil(t, workloads)
	assert.Equal(t, 2, len(workloads))
	assert.NoError(t, err)
}

func TestHandler_DeletePod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	ctx := context.Background()

	k8s := k8smocks.NewMockKubernetesManager(t)
	kv := kvmocks.NewMockKubevirtManager(t)

	h := NewHandler(testConfigPath, k8s, kv)
	assert.NotNil(t, h)
	k8s.On("DeletePod", ctx, testCluster1, testNamespace, testPodName).Return(nil)

	err := h.DeletePod(ctx, testCluster1, testNamespace, testPodName)
	assert.NoError(t, err)

}
