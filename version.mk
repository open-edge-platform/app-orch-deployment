# version.mk - check versions of tools for App Orchestration modules

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
 
GOLINTVERSION_HAVE          := $(shell golangci-lint version | sed 's/.*version //' | sed 's/ .*//')
GOLINTVERSION_REQ           := 1.61.0
GOJUNITREPORTVERSION_HAVE   := $(shell go-junit-report -version | sed s/.*" v"// | sed 's/ .*//')
GOJUNITREPORTVERSION_REQ    := 2.1.0
OPAVERSION_HAVE             := $(shell opa version | grep "Version:" | grep -v "Go" | sed 's/.*Version: //')
OPAVERSION_REQ              := 0.69.0
GOVERSION_REQ               := 1.23.2
GOVERSION_HAVE              := $(shell go version | sed 's/.*version go//' | sed 's/ .*//')
MOCKGENVERSION_HAVE         := $(shell mockgen -version | sed s/.*"v"// | sed 's/ .*//')
MOCKGENVERSION_REQ          := 1.6.0
OAPI_CODEGEN_VERSION_HAVE   := $(shell oapi-codegen -version | sed -n 2p | sed s/v//)
OAPI_CODEGEN_VERSION_REQ    := 2.3.0
ATLAS_REQ                   := $(shell command -v atlas)
GCC_REQ                     := $(shell command -v gcc)
PROTOCGENDOCVERSION_HAVE    := $(shell protoc-gen-doc --version | sed s/.*"version "// | sed 's/ .*//')
PROTOCGENDOCVERSION_REQ     := 1.5.1
BUFVERSION_HAVE             := $(shell buf --version)
BUFVERSION_REQ              := 1.45.0
PROTOCGENGOGRPCVERSION_HAVE := $(shell protoc-gen-go-grpc -version | sed s/.*"protoc-gen-go-grpc "// | sed 's/ .*//')
PROTOCGENGOGRPCVERSION_REQ  := 1.2.0
PROTOCGENGOVERSION_HAVE     := $(shell protoc-gen-go --version | sed s/.*"protoc-gen-go v"// | sed 's/ .*//')
PROTOCGENGOVERSION_REQ      := 1.30.0

# No version reported
GOCOBERTURAVERSION_REQ      := 1.2.0
PROTOCGENENTVERSION_REQ     := 0.6.0

go-dependency-check:
	@(echo "$(GOVERSION_HAVE)" | grep "$(GOVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of go\nRecommended: $(GOVERSION_REQ)\nYours: $(GOVERSION_HAVE)\e[1;m" && exit 1)
ifeq ($(GCC), true)
	@(if ! [ $(GCC_REQ) > /dev/null 2>&1 ]; then echo "\e[1;31mWARNING: You seem not having \"gcc\" installed\e[1;m" && exit 1 ; fi)
endif
ifeq ($(GOLINT), true)
	@(echo "$(GOLINTVERSION_HAVE)" | grep "$(GOLINTVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of go-lint\nRecommended: $(GOLINTVERSION_REQ)\nYours: $(GOLINTVERSION_HAVE)\e[1;m" && exit 1)
endif
ifeq ($(GOJUNITREPORT), true)
	@(echo "$(GOJUNITREPORTVERSION_HAVE)" | grep "$(GOJUNITREPORTVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of go-junit-report\nRecommended: $(GOJUNITREPORTVERSION_REQ)\nYours: $(GOJUNITREPORTVERSION_HAVE)\e[1;m" && exit 1)
endif
ifeq ($(OAPI_CODEGEN), true)
	@(echo "$(OAPI_CODEGEN_VERSION_HAVE)" | grep "$(OAPI_CODEGEN_VERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of oapi-codegen\nRecommended: $(OAPI_CODEGEN_VERSION_REQ)\nYours: $(OAPI_CODEGEN_VERSION_HAVE)\e[1;m" && exit 1)
endif
ifeq ($(OPA), true)
	@(echo "$(OPAVERSION_HAVE)" | grep "$(OPAVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of opan\nRecommended: $(OPAVERSION_REQ)\nYours: $(OPAVERSION_HAVE)\e[1;m" && exit 1)
endif
ifeq ($(MOCKGENGEN), true)
	@(echo "$(MOCKGENVERSION_HAVE)" | grep "$(MOCKGENVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of mockgen\nRecommended:: $(MOCKGENVERSION_REQ)"\nYours: $(MOCKGENVERSION_HAVE)\e[1;m" && exit 1)
endif
ifeq ($(ATLAS), true)
	@(if ! [ $(ATLAS_REQ) > /dev/null 2>&1 ]; then echo "\e[1;31mWARNING: You seem not having \"atlas\" installed\e[1;m" && exit 1 ; fi)
endif
ifeq ($(PROTOCGENDOC), true)
	@(echo "$(PROTOCGENDOCVERSION_HAVE)" | grep "$(PROTOCGENDOCVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of protoc-gen-doc\nRecommended: $(PROTOCGENDOCVERSION_REQ)\nYours: $(PROTOCGENDOCVERSION_HAVE)\e[1;m" && exit 1)
endif
ifeq ($(BUF), true)
	@(echo "$(BUFVERSION_HAVE)" | grep "$(BUFVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of buf\nRecommended: $(BUFVERSION_REQ)\nYours: $(BUFVERSION_HAVE)\e[1;m" && exit 1)
endif
ifeq ($(PROTOCGENGO), true)
	@(echo "$(PROTOCGENGOVERSION_HAVE)" | grep "$(PROTOCGENGOVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of protoc-gen-go\nRecommended: $(PROTOCGENGOVERSION_REQ)\nYours: $(PROTOCGENGOVERSION_HAVE)\e[1;m" && exit 1)
endif
ifeq ($(PROTOCGENGOGRPC), true)
	@(echo "$(PROTOCGENGOGRPCVERSION_HAVE)" | grep "$(PROTOCGENGOGRPCVERSION_REQ)" > /dev/null) || \
	(echo  "\e[1;31mWARNING: You are not using the recommended version of protoc-gen-go-grpc\nRecommended: $(PROTOCGENGOGRPCVERSION_REQ)\nYours: $(PROTOCGENGOGRPCVERSION_HAVE)\e[1;m" && exit 1)
endif

go-dependency: ## install go dependency tooling
ifeq ($(OAPI_CODEGEN), true)
	${GOCMD} install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v${OAPI_CODEGEN_VERSION_REQ}
endif
ifeq ($(GOJUNITREPORT), true)
	${GOCMD} install github.com/jstemmer/go-junit-report/v2@v$(GOJUNITREPORTVERSION_REQ)
endif
ifeq ($(GOLINT), true)
	${GOCMD} install github.com/golangci/golangci-lint/cmd/golangci-lint@v${GOLINTVERSION_REQ}
endif
ifeq ($(PROTOCGENENT), true)
	$(GOCMD) install entgo.io/contrib/entproto/cmd/protoc-gen-ent@v$(PROTOCGENENTVERSION_REQ)
endif
ifeq ($(MOCKGEN), true)
	$(GOCMD) install github.com/golang/mock/mockgen@v$(MOCKGENVERSION_REQ)
endif
ifeq ($(BUF), true)
	$(GOCMD) install github.com/bufbuild/buf/cmd/buf@v${BUFVERSION_REQ}
endif
ifeq ($(PROTOCGENDOC), true)
	$(GOCMD) install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@v${PROTOCGENDOCVERSION_REQ}
endif
ifeq ($(GOCOBERTURA), true)
	${GOCMD} install github.com/boumenot/gocover-cobertura@v$(GOCOBERTURAVERSION_REQ)
endif
ifeq ($(PROTOCGENGO), true)
	$(GOCMD) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${PROTOCGENGOGRPCVERSION_REQ}
endif
ifeq ($(PROTOCGENGOGRPC), true)
	$(GOCMD) install google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOCGENGOVERSION_REQ}
endif
