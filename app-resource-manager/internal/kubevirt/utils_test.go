// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubevirt

import (
	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"testing"
)

func getMockK8sRESTConfigFromKubeConfig(outputCfg *rest.Config, outputErr error) func(configBytes []byte) (*rest.Config, error) {
	return func(configBytes []byte) (*rest.Config, error) {
		return outputCfg, outputErr
	}
}

func getMockKubeVirtGetKubevirtClientFromRESTConfig(outputKubeVirtClient kubecli.KubevirtClient, outputErr error) func(config *rest.Config) (kubecli.KubevirtClient, error) {
	return func(config *rest.Config) (kubecli.KubevirtClient, error) {
		return outputKubeVirtClient, outputErr
	}
}

func TestConvertVMStatusV2(t *testing.T) {
	var result *resourceapiv2.VirtualMachineStatus
	result = convertVMStatusV2(v1.VirtualMachineStatusStopped)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_STOPPED, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusProvisioning)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_PROVISIONING, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusStarting)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_STARTING, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusRunning)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_RUNNING, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusPaused)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_PAUSED, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusStopping)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_STOPPING, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusTerminating)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_TERMINATING, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusCrashLoopBackOff)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_CRASH_LOOP_BACKOFF, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusMigrating)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_MIGRATING, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusUnknown)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_UNSPECIFIED, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusUnschedulable)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_ERROR_UNSCHEDULABLE, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusErrImagePull)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_ERROR_IMAGE_PULL, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusImagePullBackOff)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_ERROR_IMAGE_PULL_BACKOFF, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusPvcNotFound)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_ERROR_PVC_NOT_FOUND, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusDataVolumeError)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_ERROR_DATA_VOLUME, result.State)
	result = convertVMStatusV2(v1.VirtualMachineStatusWaitingForVolumeBinding)
	assert.Equal(t, resourceapiv2.VirtualMachineStatus_STATE_WAITING_FOR_VOLUME_BINDING, result.State)
}

func TestConvertAdminStatusV2(t *testing.T) {
	runningStateTrue := true
	runningStateFalse := false
	var result *resourceapiv2.AdminStatus
	result = convertAdminStatusV2(&runningStateTrue)
	assert.Equal(t, resourceapiv2.AdminStatus_STATE_UP, result.State)
	result = convertAdminStatusV2(&runningStateFalse)
	assert.Equal(t, resourceapiv2.AdminStatus_STATE_DOWN, result.State)
}
