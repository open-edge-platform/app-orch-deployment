// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package types //nolint:revive // types is an appropriate name for this package

import (
	"os"
	"time"
)

const (
	RestAddressPortForward      = "127.0.0.1"
	PortForwardServiceNamespace = "orch-app"
	AdmPortForwardService       = "svc/app-deployment-api-rest-proxy"
	ArmPortForwardService       = "svc/app-resource-manager-rest-proxy"
	AdmPortForwardLocal         = "8081"
	ArmPortForwardLocal         = "8081"
	PortForwardAddress          = "0.0.0.0"
	AdmPortForwardRemote        = "8081"
	ArmPortForwardRemote        = "8082"
)

const (
	RetryDelay = 10 * time.Second
	RetryCount = 20
)

const (
	SampleOrg     = "sample-org"
	SampleProject = "sample-project"
)

// TestClusterID is the cluster ID used for testing.
// It reads from TEST_CLUSTER_ID environment variable, defaults to "demo-cluster".
var TestClusterID = getTestClusterID()

var KCPass = mustGetKCPassword()

// getTestClusterID returns the cluster ID from environment variable or default.
func getTestClusterID() string {
	clusterID := os.Getenv("TEST_CLUSTER_ID")
	if clusterID == "" {
		return "demo-cluster"
	}
	return clusterID
}

func mustGetKCPassword() string {
	pass := os.Getenv("ORCH_DEFAULT_PASSWORD")
	if pass == "" {
		panic("ORCH_DEFAULT_PASSWORD environment variable must be set")
	}
	return pass
}
