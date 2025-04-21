// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

module github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2

go 1.23.0

toolchain go1.24.1

require (
	github.com/envoyproxy/protoc-gen-validate v1.0.4
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0
	github.com/oapi-codegen/runtime v1.1.1
	google.golang.org/genproto/googleapis/api v0.0.0-20240624140628-dc46fd24d27d
	google.golang.org/grpc v1.64.1
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240617180043-68d350f18fd4 // indirect
)
