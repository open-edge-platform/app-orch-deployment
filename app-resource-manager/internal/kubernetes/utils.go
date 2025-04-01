// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"

	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/utils/ratelimiter"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, _ string) (kubernetes.Interface, error) {
	kubeConfig, err := admClient.GetKubeConfig(ctx, clusterID)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	kubeRESTConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
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

	clientSet, err := kubernetes.NewForConfig(kubeRESTConfig)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	return clientSet, nil
}

var getDynamicK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, _ string) (dynamic.Interface, error) {
	kubeConfig, err := admClient.GetKubeConfig(ctx, clusterID)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	kubeRESTConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
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

	clientSet, err := dynamic.NewForConfig(kubeRESTConfig)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	return clientSet, nil
}

func convertPodPhase(phase corev1.PodPhase) *resourceapiv2.PodStatus {
	state := resourceapiv2.PodStatus_STATE_UNSPECIFIED
	switch phase {
	case corev1.PodRunning:
		state = resourceapiv2.PodStatus_STATE_RUNNING
	case corev1.PodFailed:
		state = resourceapiv2.PodStatus_STATE_FAILED
	case corev1.PodPending:
		state = resourceapiv2.PodStatus_STATE_PENDING
	case corev1.PodSucceeded:
		state = resourceapiv2.PodStatus_STATE_SUCCEEDED
	}

	return &resourceapiv2.PodStatus{
		State: state,
	}
}

func convertContainerStatus(containerState corev1.ContainerState) *resourceapiv2.ContainerStatus {
	status := &resourceapiv2.ContainerStatus{}
	switch {
	case containerState.Waiting != nil:
		state := &resourceapiv2.ContainerStateWaiting{
			Reason:  containerState.Waiting.Reason,
			Message: containerState.Waiting.Message,
		}
		status.State = &resourceapiv2.ContainerStatus_ContainerStateWaiting{
			ContainerStateWaiting: state,
		}
	case containerState.Running != nil:
		state := &resourceapiv2.ContainerStateRunning{}
		status.State = &resourceapiv2.ContainerStatus_ContainerStateRunning{
			ContainerStateRunning: state,
		}
	case containerState.Terminated != nil:
		state := &resourceapiv2.ContainerStateTerminated{
			ExitCode: containerState.Terminated.ExitCode,
			Reason:   containerState.Terminated.Reason,
			Message:  containerState.Terminated.Message,
		}
		status.State = &resourceapiv2.ContainerStatus_ContainerStateTerminated{
			ContainerStateTerminated: state,
		}
	}
	return status

}

func createServiceProxyURL(svcProxyURL serviceProxyURL) string {
	url := ""
	if svcProxyURL.serviceProtocol == "" {
		url = fmt.Sprintf("%s/app-service-proxy-index.html?project=%s&cluster=%s&namespace=%s&service=%s:%s",
			svcProxyURL.domainName, svcProxyURL.projectID, svcProxyURL.clusterID, svcProxyURL.serviceNamespace,
			svcProxyURL.serviceName, svcProxyURL.servicePort)
	} else {
		url = fmt.Sprintf("%s/app-service-proxy-index.html?project=%s&cluster=%s&namespace=%s&service=%s:%s:%s",
			svcProxyURL.domainName, svcProxyURL.projectID, svcProxyURL.clusterID, svcProxyURL.serviceNamespace,
			svcProxyURL.serviceProtocol, svcProxyURL.serviceName, svcProxyURL.servicePort)
	}

	return url
}
