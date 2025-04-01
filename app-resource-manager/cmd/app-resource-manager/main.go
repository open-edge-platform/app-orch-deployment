// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/manager"
	_ "github.com/open-edge-platform/orch-library/go/dazl/zap"
)

// The main entry point
func main() {
	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")
	configPath := flag.String("configPath", "/opt/app-resource-manager/config.yaml", "path to config file")
	flag.Parse()

	ready := make(chan bool)
	cfg := manager.Config{
		CAPath:     *caPath,
		KeyPath:    *keyPath,
		CertPath:   *certPath,
		GRPCPort:   8080,
		WSPort:     5900,
		ConfigPath: *configPath,
	}

	mgr := manager.NewManager(cfg)
	mgr.Run()
	<-ready
}
