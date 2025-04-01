// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"testing"
)

func TestConvertPodPhase(t *testing.T) {
	podStatus := convertPodPhase(corev1.PodRunning)
	assert.Equal(t, resourceapiv2.PodStatus_STATE_RUNNING, podStatus.State)
	podStatus = convertPodPhase(corev1.PodPending)
	assert.Equal(t, resourceapiv2.PodStatus_STATE_PENDING, podStatus.State)
	podStatus = convertPodPhase(corev1.PodFailed)
	assert.Equal(t, resourceapiv2.PodStatus_STATE_FAILED, podStatus.State)
	podStatus = convertPodPhase(corev1.PodSucceeded)
	assert.Equal(t, resourceapiv2.PodStatus_STATE_SUCCEEDED, podStatus.State)

}

func TestConvertContainerStatus(t *testing.T) {
	containerStatus := convertContainerStatus(corev1.ContainerState{
		Waiting: &corev1.ContainerStateWaiting{
			Reason:  "wait-reason",
			Message: "wait-message",
		},
	})
	assert.Equal(t, &resourceapiv2.ContainerStateWaiting{Reason: "wait-reason", Message: "wait-message"}, containerStatus.GetContainerStateWaiting())
	containerStatus = convertContainerStatus(corev1.ContainerState{
		Running: &corev1.ContainerStateRunning{},
	})
	assert.Equal(t, &resourceapiv2.ContainerStateRunning{}, containerStatus.GetContainerStateRunning())
	containerStatus = convertContainerStatus(corev1.ContainerState{
		Terminated: &corev1.ContainerStateTerminated{
			ExitCode: 0,
		},
	})
	assert.Equal(t, &resourceapiv2.ContainerStateTerminated{
		ExitCode: 0,
	}, containerStatus.GetContainerStateTerminated())
}

func TestCreateServiceProxyURL(t *testing.T) {
	cases := []struct {
		svcProxyURL serviceProxyURL
		expectedURL string
	}{
		{
			svcProxyURL: serviceProxyURL{
				domainName:       "kind.internal",
				projectID:        "test",
				serviceNamespace: "testnamespace",
				serviceName:      "testservice",
				servicePort:      "80",
				serviceProtocol:  "https",
				clusterID:        "testcluster",
			},
			expectedURL: "kind.internal/app-service-proxy-index.html?project=test&cluster=testcluster&namespace=testnamespace&service=https:testservice:80",
		},
		{
			svcProxyURL: serviceProxyURL{
				domainName:       "kind.internal",
				projectID:        "test",
				serviceNamespace: "testnamespace",
				serviceName:      "testservice",
				servicePort:      "80",
				clusterID:        "testcluster",
			},
			expectedURL: "kind.internal/app-service-proxy-index.html?project=test&cluster=testcluster&namespace=testnamespace&service=testservice:80",
		},
		{
			svcProxyURL: serviceProxyURL{
				domainName:       "kind.internal",
				projectID:        "test",
				serviceNamespace: "testnamespace",
				serviceName:      "testservice",
				servicePort:      "80",
				serviceProtocol:  "http",
				clusterID:        "testcluster",
			},
			expectedURL: "kind.internal/app-service-proxy-index.html?project=test&cluster=testcluster&namespace=testnamespace&service=http:testservice:80",
		},
	}

	for _, c := range cases {
		actual := createServiceProxyURL(c.svcProxyURL)
		assert.Equal(t, c.expectedURL, actual)
	}

}
