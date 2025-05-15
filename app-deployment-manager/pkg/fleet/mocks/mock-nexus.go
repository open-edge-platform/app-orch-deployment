// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleetclient

import (
	"context"
	"fmt"
	v1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimeproject.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/client/clientset/versioned/typed/runtimeproject.edge-orchestrator.intel.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type MockRuntimeProjects struct {
	runtimeProjects v1.RuntimeProjectList
}

func (m *MockRuntimeProjects) Create(ctx context.Context, runtimeProject *v1.RuntimeProject, opts metav1.CreateOptions) (*v1.RuntimeProject, error) {
	_ = ctx
	_ = opts
	_ = runtimeProject
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRuntimeProjects) Update(ctx context.Context, runtimeProject *v1.RuntimeProject, opts metav1.UpdateOptions) (*v1.RuntimeProject, error) {
	_ = ctx
	_ = opts
	_ = runtimeProject
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRuntimeProjects) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_ = ctx
	_ = name
	_ = opts
	return fmt.Errorf("not implemented")
}

func (m *MockRuntimeProjects) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	_ = ctx
	_ = opts
	_ = listOpts
	return fmt.Errorf("not implemented")
}

func (m *MockRuntimeProjects) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.RuntimeProject, error) {
	_ = ctx
	_ = opts
	for _, project := range m.runtimeProjects.Items {
		if project.Name == name {
			return &project, nil
		}
	}
	return nil, fmt.Errorf("project %s not found", name)
}

func (m *MockRuntimeProjects) List(ctx context.Context, opts metav1.ListOptions) (*v1.RuntimeProjectList, error) {
	_ = ctx
	_ = opts
	return &m.runtimeProjects, nil
}

func (m *MockRuntimeProjects) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	_ = ctx
	_ = opts
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRuntimeProjects) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.RuntimeProject, err error) {
	_ = ctx
	_ = name
	_ = pt
	_ = data
	_ = opts
	_ = subresources
	return nil, fmt.Errorf("not implemented")
}

type MockNexusClient struct {
	mockRuntimeProjects *MockRuntimeProjects
}

func (nc *MockNexusClient) RuntimeProjects() nexus.RuntimeProjectInterface {
	return nc.mockRuntimeProjects
}

func (nc *MockNexusClient) RESTClient() rest.Interface {
	return nil
}

func NewMockNexusClient(runtimeProjects []v1.RuntimeProject) nexus.RuntimeprojectEdgeV1Interface {
	return &MockNexusClient{
		mockRuntimeProjects: &MockRuntimeProjects{
			runtimeProjects: v1.RuntimeProjectList{
				Items: runtimeProjects,
			},
		},
	}
}
