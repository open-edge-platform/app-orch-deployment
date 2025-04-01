// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/manager"
)

func main() {
	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")
	kubeconfig := flag.String("kubeconfig", "", "path to kubeconfig")
	flag.Parse()

	ready := make(chan bool)
	cfg := manager.Config{
		CAPath:     *caPath,
		KeyPath:    *keyPath,
		CertPath:   *certPath,
		GRPCPort:   8080,
		Kubeconfig: *kubeconfig,
	}

	mgr := manager.NewManager(cfg)
	mgr.Run()
	<-ready
}
