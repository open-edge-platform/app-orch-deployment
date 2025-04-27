// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var log = dazl.GetPackageLogger()

var (
	// Custom collector
	Reg       = prometheus.NewRegistry()
	TimingReg = prometheus.NewRegistry()

	TimestampGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "code_execution_timestamp",
			Help: "Timestamp of code execution in different parts of the code",
		},
		[]string{"projectID", "deploymentID", "part", "event"},
	)

	TimeDifferenceGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "code_execution_time_difference",
			Help: "Time difference between first and last timestamp",
		},
		[]string{"projectID", "deploymentID", "firstKey", "lastKey"},
	)

	// Map to store timestamps for each deployment
	Timestamps = make(map[string]map[string]float64)

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

// RunMetricsServer starts an HTTP server to expose Prometheus metrics
func RunMetricsServer(port int16) {
	// Set up the HTTP handler for the /metrics endpoint
	http.Handle("/timing", promhttp.HandlerFor(TimingReg, promhttp.HandlerOpts{}))

	// Start the HTTP server
	addr := fmt.Sprintf(":%d", port)
	log.Infof("Starting metrics server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Errorf("Error starting metrics server: %v\n", err)
	}
}

func init() {
	// Register custom metrics with prometheus registry
	log.Infof("metrics server init \n")
	Reg.MustRegister(DeploymentStatus, DeploymentClusterStatus)
	TimingReg.MustRegister(TimestampGauge, TimeDifferenceGauge)
}
