// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubernetes"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubevirt"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
)

var log = dazl.GetPackageLogger()

// NewHandler returns a southbound handler
func NewHandler(configPath string, kubernetesManager kubernetes.Manager, kubevirtManager kubevirt.Manager) Handler {
	return &handler{
		configPath:        configPath,
		kubernetesManager: kubernetesManager,
		kubevirtManager:   kubevirtManager,
	}
}

//go:generate mockery --name Handler --filename handler_mock.go --structname MockHandler
type Handler interface {
	// StartVM starts the specific VM
	StartVM(ctx context.Context, appID string, clusterID string, vmID string) error
	// StopVM stops the specific VM
	StopVM(ctx context.Context, appID string, clusterID string, vmID string) error
	// RestartVM restarts the specific VM
	RestartVM(ctx context.Context, appID string, clusterID string, vmID string) error
	// AccessVMWithVNC returns VNC Session WebSocket address
	AccessVMWithVNC(ctx context.Context, appID string, clusterID string, vmID string) (string, error)
	// GetAppEndpointsV2 returns the endpoints that are exposed by the applications for external access using v2 API
	GetAppEndpointsV2(ctx context.Context, appID string, clusterID string) ([]*resourceapiv2.AppEndpoint, error)
	// GetAppWorkloads returns a list of application workloads
	GetAppWorkLoads(ctx context.Context, appID string, clusterID string) ([]*resourceapiv2.AppWorkload, error)
	// DeletePod deletes a pod based on a given pod name
	DeletePod(ctx context.Context, clusterID string, namespace string, podName string) error
}

// handler is a struct for a southbound handler
type handler struct {
	configPath        string             // configPath is the path of ARM configuration file
	kubernetesManager kubernetes.Manager // kubernetesManager is the manager to call Kubernetes API and handle its result
	kubevirtManager   kubevirt.Manager   // kubevirtManager is the manager to call Kubevirt API and handle its result
}

func (s *handler) DeletePod(ctx context.Context, clusterID string, namespace string, podName string) error {
	log.Debugw("Processing delete pod request", dazl.String("podName", podName))
	return s.kubernetesManager.DeletePod(ctx, clusterID, namespace, podName)

}

func (s *handler) GetAppWorkLoads(ctx context.Context, appID string, clusterID string) ([]*resourceapiv2.AppWorkload, error) {
	log.Debugw("Processing Get App Workload Request", dazl.String("appID", appID),
		dazl.String("clusterID", clusterID))
	appWorkloads := make([]*resourceapiv2.AppWorkload, 0)

	vmWorkLoads, err := s.kubevirtManager.GetVMWorkloads(ctx, appID, clusterID)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	if len(vmWorkLoads) != 0 {
		appWorkloads = append(appWorkloads, vmWorkLoads...)
	}

	podWorkLoads, err := s.kubernetesManager.GetPodWorkloads(ctx, appID, clusterID)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	if len(podWorkLoads) != 0 {
		appWorkloads = append(appWorkloads, podWorkLoads...)
	}

	return appWorkloads, nil

}

func (s *handler) StartVM(ctx context.Context, appID string, clusterID string, vmID string) error {
	log.Debugw("Processing Start VM Request", dazl.String("appID", appID),
		dazl.String("clusterID", clusterID),
		dazl.String("vmID", vmID))
	return s.kubevirtManager.StartVM(ctx, appID, clusterID, vmID)
}

func (s *handler) StopVM(ctx context.Context, appID string, clusterID string, vmID string) error {
	log.Debugw("Processing Stop VM Request", dazl.String("appID", appID),
		dazl.String("clusterID", clusterID),
		dazl.String("vmID", vmID))
	return s.kubevirtManager.StopVM(ctx, appID, clusterID, vmID)
}

func (s *handler) RestartVM(ctx context.Context, appID string, clusterID string, vmID string) error {
	log.Debugw("Processing Restart VM Request", dazl.String("appID", appID),
		dazl.String("clusterID", clusterID),
		dazl.String("vmID", vmID))
	return s.kubevirtManager.RestartVM(ctx, appID, clusterID, vmID)
}

func (s *handler) AccessVMWithVNC(ctx context.Context, appID string, clusterID string, vmID string) (string, error) {
	log.Debugw("Processing Accessing VM with VNC", dazl.String("appID", appID),
		dazl.String("clusterID", clusterID), dazl.String("vmID", vmID))
	return s.kubevirtManager.GetVNCAddress(ctx, appID, clusterID, vmID)
}

func (s *handler) GetAppEndpointsV2(ctx context.Context, appID string, clusterID string) ([]*resourceapiv2.AppEndpoint, error) {
	log.Debugw("Processing Get App Endpoints request", dazl.String("appID", appID),
		dazl.String("clusterID", clusterID))
	return s.kubernetesManager.GetAppEndpointsV2(ctx, appID, clusterID)
}
