// SPDX-FileCopyrightText: 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import "time"

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
	SampleOrg     = "sample-org"
	SampleProject = "sample-project"
	DefaultPass   = "ChangeMeOn1stLogin!"
)

const InvalidJWT = "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.ey" +
	"JzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRt" +
	"aW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.J5W09-rNx0pt5_HBiy" +
	"dR-vOluS6oD-RpYNa8PVWwMcBDQSXiw6-EPW8iSsalXPspGj3ouQjA" +
	"nOP_4-zrlUUlvUIt2T79XyNeiKuooyIFvka3Y5NnGiOUBHWvWcWp4R" +
	"cQFMBrZkHtJM23sB5D7Wxjx0-HFeNk-Y3UJgeJVhg5NaWXypLkC4y0" +
	"ADrUBfGAxhvGdRdULZivfvzuVtv6AzW6NRuEE6DM9xpoWX_4here-y" +
	"vLS2YPiBTZ8xbB3axdM99LhES-n52lVkiX5AWg2JJkEROZzLMpaacA" +
	"_xlbUz_zbIaOaoqk8gB5oO7kI6sZej3QAdGigQy-hXiRnW_L98d4GQ"

const (
	TestClusterID                  = "demo-cluster"
	retryCount                     = 10
	NginxAppName                   = "nginx"
	CirrosAppName                  = "cirros-container-disk"
	WordpressAppName               = "wordpress"
	VirtualizationExtensionAppName = "virtualization-extension"
)
const (
	retryDelay = 10 * time.Second
)

var DpConfigs = map[string]any{
	"nginx": map[string]any{
		"appNames":             []string{"nginx"},
		"deployPackage":        "nginx-app",
		"deployPackageVersion": "0.1.0",
		"profileName":          "testing-default",
		"clusterId":            TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
	"vm": map[string]any{
		"appNames":             []string{"librespeed-vm"},
		"deployPackage":        "librespeed-app",
		"deployPackageVersion": "1.0.0",
		"profileName":          "virtual-cluster",
		"clusterId":            TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
	"virt-extension": map[string]any{
		"appNames":             []string{"kubevirt", "cdi", "kube-helper"},
		"deployPackage":        "virtualization",
		"deployPackageVersion": "0.3.7",
		"profileName":          "with-software-emulation-profile-nosm",
		"clusterId":            TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
	"wordpress": map[string]any{
		"appNames":             []string{"wordpress"},
		"deployPackage":        "wordpress",
		"deployPackageVersion": "0.1.0",
		"profileName":          "testing",
		"clusterId":            TestClusterID,
		"labels":               map[string]string{"color": "blue"},
		"overrideValues":       []map[string]any{},
	},
}
