// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubevirt

import (
	"context"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/utils/k8serrors"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func (m *manager) getKubevirtVirtualMachineList(ctx context.Context, client kubecli.KubevirtClient, appID string) ([]*v1.VirtualMachine, error) {
	results := make([]*v1.VirtualMachine, 0)
	appNamespace, err := m.admClient.GetAppNamespace(ctx, appID)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	vmList, err := client.VirtualMachine(appNamespace).List(ctx, &k8sv1.ListOptions{})
	if err != nil {
		log.Warn(err)
		return nil, k8serrors.K8sToTypedError(err)
	}

	for i := 0; i < len(vmList.Items); i++ {
		vmAppID := ""
		if _, ok := vmList.Items[i].Annotations[AnnotationKeyForAppID]; ok {
			vmAppID = vmList.Items[i].Annotations[AnnotationKeyForAppID]
		}

		if vmAppID != appID {
			continue
		}

		results = append(results, &vmList.Items[i])
	}

	return results, nil
}

func (m *manager) getKubevirtVirtualMachine(ctx context.Context, client kubecli.KubevirtClient, appID string, vmID string) (*v1.VirtualMachine, error) {
	vmList, err := m.getKubevirtVirtualMachineList(ctx, client, appID)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(vmList); i++ {
		id := string(vmList[i].ObjectMeta.UID)
		if id == vmID {
			return vmList[i], nil
		}
	}

	return nil, errors.NewNotFound("failed to get VM (appID %s, vmID %s)", appID, vmID)
}
