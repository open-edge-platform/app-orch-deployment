// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/ratelimiter"

	"github.com/open-edge-platform/orch-library/go/dazl"
	"k8s.io/client-go/kubernetes"
)

var log = dazl.GetPackageLogger()

// NewClient returns an instance of k8s client based on a given kubeConfig
func NewClient(kubeConfig string) (*kubernetes.Clientset, error) {
	config, err := utils.CreateRestConfig(kubeConfig)
	if err != nil {
		log.Warnw("Failed to create REST config from kubeConfig", dazl.Error(err))
		return nil, err
	}

	qps, burst, err := ratelimiter.GetRateLimiterParams()
	if err != nil {
		log.Warnw("Failed to get rate limiter parameters", dazl.Error(err))
		return nil, err
	}

	config.QPS = float32(qps)
	config.Burst = int(burst)

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Warnw("Failed to create k8s clientset", dazl.Error(err))
		return nil, err
	}

	return clientSet, nil
}
