// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package types

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
	TestClusterID = "demo-cluster"
)

var KCPass = mustGetKCPassword()

func mustGetKCPassword() string {
	pass := os.Getenv("ORCH_DEFAULT_PASSWORD")
	if pass == "" {
		panic("ORCH_DEFAULT_PASSWORD environment variable must be set")
	}
	return pass
}
