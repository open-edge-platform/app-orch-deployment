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
	grpcAddr           string
	gwAddr             int
	metricsPort        int
	allowedCorsOrigins string
	basePath           string
}

func parseFlags() flags {
	f := flags{}
	flag.StringVar(&f.grpcAddr, "grpcAddr", "localhost:8080", "The endpoint of the gRPC server")
	flag.IntVar(&f.gwAddr, "gwAddr", 8081, "port that REST service runs on")
	flag.StringVar(&f.allowedCorsOrigins, "allowedCorsOrigins", "", "Comma separated list of allowed CORS origins")
	flag.StringVar(&f.basePath, "basePath", "", "The rest server base Path")
	flag.IntVar(&f.metricsPort, "metricsPort", 8082, "The port the metric endpoint binds to.")

	flag.Parse()

	return f
}

func main() {
	f := parseFlags()

	log.Infof("Serving gRPC-Gateway on port %d", f.gwAddr)

	err := restproxy.Run(f.grpcAddr, f.gwAddr, f.allowedCorsOrigins, f.basePath,
		"/usr/local/etc/openapi.yaml", f.metricsPort)
	if err != nil {
		log.Fatalf("Failed to run gateway server %v", err)
	}
}
