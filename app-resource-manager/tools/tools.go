// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build tools

// Purpose: This file ensures that dependent Go command-line tools are installed.
//
// Use: Run `go mod tidy` to install dependent Go command-line tools.
//
// Docs: Go tool dependencies
//
//	https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
//
// Docs: Go build constraints
//
//	https://pkg.go.dev/go/build#hdr-Build_Constraints
//
// Docs: gnostic/protoc-gen-openapi
//
//	https://github.com/google/gnostic/blob/main/cmd/protoc-gen-openapi/README.md
//
// Manually install tools if needed.
//
//	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
//	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
//	go install entgo.io/contrib/entproto/cmd/protoc-gen-ent@latest
//	go install entgo.io/ent/cmd/ent@latest
package tools

import (
	// _ "github.com/google/gnostic/cmd/protoc-gen-openapi"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/lib/pq"

	_ "entgo.io/contrib/entproto/cmd/protoc-gen-ent"
	_ "entgo.io/ent/cmd/ent"
	_ "github.com/envoyproxy/protoc-gen-validate"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
