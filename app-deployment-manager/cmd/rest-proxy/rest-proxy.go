// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/restproxy"
	"github.com/open-edge-platform/orch-library/go/dazl"
	_ "github.com/open-edge-platform/orch-library/go/dazl/zap"
)

var log = dazl.GetPackageLogger()

type flags struct {
	backendAddr        string
	gwAddr             int
	allowedCorsOrigins string
	basePath           string
	nexusAPIURL        string
}

func parseFlags() flags {
	f := flags{}
	flag.StringVar(&f.backendAddr, "backendAddr", "localhost:8080", "The endpoint of the backend server")
	flag.IntVar(&f.gwAddr, "gwAddr", 8081, "port that REST service runs on")
	flag.StringVar(&f.allowedCorsOrigins, "allowedCorsOrigins", "", "Comma separated list of allowed CORS origins")
	flag.StringVar(&f.basePath, "basePath", "", "The rest server base Path")
	flag.StringVar(&f.nexusAPIURL, "nexus-api-url", "", "URL of the Nexus API for project name to UUID resolution (e.g. http://svc-iam-nexus-api-gw.orch-iam.svc:8082)")

	flag.Parse()

	return f
}

func main() {
	f := parseFlags()

	log.Infof("Serving Connect-RPC REST proxy on port %d", f.gwAddr)
	err := restproxy.RunWithOptions(f.backendAddr, f.gwAddr, f.allowedCorsOrigins, f.basePath, "/usr/local/etc/openapi.yaml", f.nexusAPIURL)
	if err != nil {
		log.Fatalf("Failed to run Connect-RPC server %v", err)
	}
}
