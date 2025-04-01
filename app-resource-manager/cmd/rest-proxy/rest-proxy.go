// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/restproxy"
	"github.com/open-edge-platform/orch-library/go/dazl"
	_ "github.com/open-edge-platform/orch-library/go/dazl/zap"
)

var log = dazl.GetPackageLogger()

func main() {
	allowedCorsOrigins := flag.String(
		"allowedCorsOrigins",
		"",
		"Comma separated list of allowed CORS origins",
	)
	basePath := flag.String(
		"basePath",
		"",
		"The rest server base Path",
	)
	restPort := flag.Int(
		"rest-port",
		8081,
		"port that REST service runs on",
	)
	grpcEndpoint := flag.String(
		"grpc-endpoint",
		"localhost:8080",
		"The endpoint of the gRPC server",
	)

	flag.Parse()
	err := restproxy.Run(*restPort, *grpcEndpoint, *basePath, *allowedCorsOrigins, "/usr/local/etc/v2/openapi.yaml")
	if err != nil {
		log.Fatal(err)
	}
}
