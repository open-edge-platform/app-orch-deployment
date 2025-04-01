// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubevirt

import (
	"context"
	"encoding/json"
	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/utils/ratelimiter"
	"github.com/open-edge-platform/orch-library/go/dazl"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"k8s.io/client-go/tools/clientcmd"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"net/http"
)

var (
	// for unit test
	k8sRESTConfigFromKubeConfig             = clientcmd.RESTConfigFromKubeConfig
	kubeVirtGetKubevirtClientFromRESTConfig = kubecli.GetKubevirtClientFromRESTConfig
)

func (m *manager) getKubevirtClient(ctx context.Context, clusterID string) (kubecli.KubevirtClient, error) {
	kubeConfig, err := m.admClient.GetKubeConfig(ctx, clusterID)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	kubeRESTConfig, err := k8sRESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	qpsValue, burstValue, err := ratelimiter.GetRateLimiterParams()
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	kubeRESTConfig.QPS = float32(qpsValue)
	kubeRESTConfig.Burst = int(burstValue)

	kubevirtClient, err := kubeVirtGetKubevirtClientFromRESTConfig(kubeRESTConfig)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	return kubevirtClient, nil

}
func convertVMStatusV2(printableStatus v1.VirtualMachinePrintableStatus) *resourceapiv2.VirtualMachineStatus {
	status := resourceapiv2.VirtualMachineStatus_STATE_UNSPECIFIED // default: unspecified
	switch printableStatus {
	case v1.VirtualMachineStatusStopped:
		status = resourceapiv2.VirtualMachineStatus_STATE_STOPPED
	case v1.VirtualMachineStatusProvisioning:
		status = resourceapiv2.VirtualMachineStatus_STATE_PROVISIONING
	case v1.VirtualMachineStatusStarting:
		status = resourceapiv2.VirtualMachineStatus_STATE_STARTING
	case v1.VirtualMachineStatusRunning:
		status = resourceapiv2.VirtualMachineStatus_STATE_RUNNING
	case v1.VirtualMachineStatusPaused:
		status = resourceapiv2.VirtualMachineStatus_STATE_PAUSED
	case v1.VirtualMachineStatusStopping:
		status = resourceapiv2.VirtualMachineStatus_STATE_STOPPING
	case v1.VirtualMachineStatusTerminating:
		status = resourceapiv2.VirtualMachineStatus_STATE_TERMINATING
	case v1.VirtualMachineStatusCrashLoopBackOff:
		status = resourceapiv2.VirtualMachineStatus_STATE_CRASH_LOOP_BACKOFF
	case v1.VirtualMachineStatusMigrating:
		status = resourceapiv2.VirtualMachineStatus_STATE_MIGRATING
	case v1.VirtualMachineStatusUnknown:
		status = resourceapiv2.VirtualMachineStatus_STATE_UNSPECIFIED // status unknown in Kubevirt - mapped to unspecified
	case v1.VirtualMachineStatusUnschedulable:
		status = resourceapiv2.VirtualMachineStatus_STATE_ERROR_UNSCHEDULABLE
	case v1.VirtualMachineStatusErrImagePull:
		status = resourceapiv2.VirtualMachineStatus_STATE_ERROR_IMAGE_PULL
	case v1.VirtualMachineStatusImagePullBackOff:
		status = resourceapiv2.VirtualMachineStatus_STATE_ERROR_IMAGE_PULL_BACKOFF
	case v1.VirtualMachineStatusPvcNotFound:
		status = resourceapiv2.VirtualMachineStatus_STATE_ERROR_PVC_NOT_FOUND
	case v1.VirtualMachineStatusDataVolumeError:
		status = resourceapiv2.VirtualMachineStatus_STATE_ERROR_DATA_VOLUME
	case v1.VirtualMachineStatusWaitingForVolumeBinding:
		status = resourceapiv2.VirtualMachineStatus_STATE_WAITING_FOR_VOLUME_BINDING
	}

	return &resourceapiv2.VirtualMachineStatus{
		State: status,
	}
}

func convertAdminStatusV2(runningStatus *bool) *resourceapiv2.AdminStatus {
	status := resourceapiv2.AdminStatus_STATE_UNSPECIFIED
	if runningStatus == nil {
		return &resourceapiv2.AdminStatus{
			State: status,
		}
	}

	switch *runningStatus {
	case true:
		status = resourceapiv2.AdminStatus_STATE_UP
	case false:
		status = resourceapiv2.AdminStatus_STATE_DOWN
	}

	return &resourceapiv2.AdminStatus{
		State: status,
	}
}

func setErrorStatus(w http.ResponseWriter, statusCode int, errMsg string) error {
	s := &spb.Status{
		Code:    int32(statusCode), // nolint:gosec
		Message: errMsg,
	}

	log.Errorw("error detected on websocket", dazl.String("status", s.String()))

	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(s)
}
