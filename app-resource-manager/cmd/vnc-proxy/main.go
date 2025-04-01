// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/vncproxy"
	_ "github.com/open-edge-platform/orch-library/go/dazl/zap"
)

const webFileDir = "/usr/local/html/vnc-proxy-web-ui"

func main() {
	configPath := flag.String("configPath", "/opt/vnc-proxy/config.yaml", "path to config file")
	flag.Parse()

	ready := make(chan bool)
	cfg := vncproxy.Config{
		WSPort:     5900,
		ConfigPath: *configPath,
		FileBase:   webFileDir,
	}

	mgr := vncproxy.NewManager(cfg)
	mgr.Run()
	<-ready
}
