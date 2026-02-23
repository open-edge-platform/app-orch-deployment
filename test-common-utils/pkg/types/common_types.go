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

// Default values for org and project names
const (
	DefaultSampleOrg     = "sample-org"
	DefaultSampleProject = "sample-project"
)

// SampleOrg returns the organization name from environment variable or default.
var SampleOrg = getSampleOrg()

// SampleProject returns the project name from environment variable or default.
var SampleProject = getSampleProject()

// SampleUsername returns the test username from environment variable or default.
var SampleUsername = getSampleUsername()

func getSampleOrg() string {
	org := os.Getenv("TEST_ORG_NAME")
	if org == "" {
		return DefaultSampleOrg
	}
	return org
}

func getSampleProject() string {
	project := os.Getenv("TEST_PROJECT_NAME")
	if project == "" {
		return DefaultSampleProject
	}
	return project
}

func getSampleUsername() string {
	username := os.Getenv("TEST_USERNAME")
	if username == "" {
		// Default format: {project}-edge-mgr
		return getSampleProject() + "-edge-mgr"
	}
	return username
}

// TestClusterID is the cluster ID used for testing.
// It reads from TEST_CLUSTER_ID environment variable, defaults to "demo-cluster".
var TestClusterID = getTestClusterID()

var KCPass = mustGetKCPassword()

// getTestClusterID returns the cluster ID from environment variable or default.
// GetTestClusterID returns the cluster ID from environment variable or default.
// This function is exported so it can be called at runtime to get the current value.
func GetTestClusterID() string {
	return getTestClusterID()
}

func getTestClusterID() string {
	clusterID := os.Getenv("TEST_CLUSTER_ID")
	if clusterID == "" {
		return "demo-cluster"
	}
	return clusterID
}

func mustGetKCPassword() string {
	// First check TEST_PASSWORD for test user credentials (used by Golden Suite)
	pass := os.Getenv("TEST_PASSWORD")
	if pass != "" {
		return pass
	}
	// Fall back to ORCH_DEFAULT_PASSWORD (orchestrator admin password)
	pass = os.Getenv("ORCH_DEFAULT_PASSWORD")
	if pass == "" {
		panic("Either TEST_PASSWORD or ORCH_DEFAULT_PASSWORD environment variable must be set")
	}
	return pass
}
