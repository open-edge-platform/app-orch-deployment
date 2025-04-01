// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Custom collector
	Reg = prometheus.NewRegistry()

	// DeploymentStatus is a prometheus metric which holds the deployment id,
	// deployment name and deployment status of a ADM per-deployment.
	DeploymentStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "adm_deployment_status",
		Help: "Per-deployment status",
	}, []string{"projectId", "deployment_id", "deployment_name", "status"})

	// DeploymentClusterStatus is a prometheus metric which holds the deployment id,
	// deployment name, cluster id and cluster status of a ADM per-deployment per-cluster.
	DeploymentClusterStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "adm_deployment_cluster_status",
		Help: "Per-deployment per-cluster status",
	}, []string{"projectId", "deployment_id", "deployment_name", "cluster_id", "cluster_name", "status"})
)

func init() {
	// Register custom metrics with prometheus registry
	Reg.MustRegister(DeploymentStatus, DeploymentClusterStatus)
}
