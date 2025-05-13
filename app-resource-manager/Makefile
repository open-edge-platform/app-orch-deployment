# Makefile for App Resource Manager

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: help build test lint

# Optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 1

##### Variables #####

# Required Go Dependencies
GOLINT        := true
GOJUNITREPORT := true
MOCKGEN       := true
OPA           := true
PROTOCGENENT  := true

# Project variables
PROJECT_NAME    := app-resource-manager
BINARY_NAME     := app-resource-manager
VNC_PROXY_NAME  := vnc-proxy
REST_PROXY_NAME := rest-proxy
CHART_PATH      := deployments/app-resource-manager

# Deployment variables
ARM_HELM_CHART_NAME	   ?= app-resource-manager
DOCKER_TAG             ?= ${VERSION}
DOCKER_REGISTRY        ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
DOCKER_REPOSITORY      ?= edge-orch
DOCKER_SUB_PROJ        ?= app
ARM_DOCKER_NAME        ?= app-resource-manager
VNC_PROXY_DOCKER_NAME  ?= app-resource-vnc-proxy
REST_PROXY_DOCKER_NAME ?= app-resource-rest-proxy
ARM_GATEWAY_NAME       ?= arm-gateway

# Coder variables
MGMT_NAME           ?= kind
MGMT_CLUSTER        ?= kind-${MGMT_NAME}
CODER_DIR           ?= ~/edge-manageability-framework
HELM_VERSION        ?= $(shell cat VERSION)
ARM_HELM_CHART      ?= app-resource-manager-${HELM_VERSION}.tgz
ARM_NS              ?= orch-app


# Helm variables
HELM_REGISTRY      ?= oci://080137407410.dkr.ecr.us-west-2.amazonaws.com
HELM_REPOSITORY    ?= edge-orch
HELM_SUB_PROJ      ?= app
HELM_CHART_PREFIX  ?= charts
HELM_CHART_NAME    ?= app-resource-manager
HELM_DIRS = $(shell find ./deployments -maxdepth 1 -mindepth 1 -type d -print )

# Versioning variables
VERSION            := $(shell cat VERSION)
CHART_VERSION      := $(shell cat VERSION)
VERSION_DEV_SUFFIX := -dev
GIT_COMMIT         ?= $(shell git rev-parse --short HEAD)
ifeq ($(patsubst %$(VERSION_DEV_SUFFIX),,$(lastword $(VERSION))),)
    DOCKER_VERSION ?= $(VERSION)-$(GIT_COMMIT)
else
    DOCKER_VERSION ?= $(VERSION)
endif

# Test variables
# Set TEST_TARGET to '<testname1>' or '<testname1\|testname2>' to run specific tests or use regex '<testname.*>' - example:
#  TEST_TARGET='TestAuth.*' make test
TEST_TARGET ?= .
# Default TEST_PKG, overwrite to select particular test pakages
TEST_PKG ?= $(shell go list ./... | grep -v "internal" | tr '\n' ,)
TEST_COVER := atomic $(shell go list ./... | grep -v "/test/")

FUZZ_SECONDS ?= 60

NO_VNC_VERSION=1.5.0
NO_VNC_RELEASES=https://github.com/novnc/noVNC/archive/refs/tags

# Include shared makefile
include ../common.mk

# Security config for Go builds
GOEXTRAFLAGS += $(COMMON_GOEXTRAFLAGS)

#### Buf protobuf code generation tooling ###

APIPKG_DIR ?= pkg/api

install-protoc-plugins: common-install-protoc-plugins ## Install protoc, buf and protobuf tools

verify-protoc-plugins: common-verify-protoc-plugins  ## Verify protoc plugins

OPENAPI_SPEC_FILE ?= ./api/nbi/v2/spec/v2/openapi.yaml
REST_CLIENT_DIR ?= ./api/nbi/v2/pkg/restClient/v2
proto-generate: common-buf-update common-buf-generate common-openapi-spec-validate common-rest-client-gen common-buf-lint ## Generate protobuf code

buf-generate: common-buf-generate ## Generate protobuf code

buf-lint-fix: common-buf-lint-fix ## Lint and format protobuf files

buf-update: common-buf-update ## Update buf modules

buf-lint: common-buf-lint ## Lint and format protobuf files

rest-client-gen: common-rest-client-gen ## Generate REST client code


#### Development Targets ####

build: go-build ## Build local binaries

# FIXME: Fix the markdown linting errors and then re-enable the mdlint tool
# FIXME: Enable hadolint for Dockerfile linting (disabled due to false positive
# when COPYing --from a named build context
lint: license helmlint go-lint lint-html lint-css lint-js #hadolint #mdlint ## Run license, helmlint, go-lint lint tools

test: go-test ## Run all unit tests

# todo: re-enable vm tests once vm extension and vm app are ready in catalog
# COMP_TEST_COVER := atomic $(shell go list ./test/... | grep -v "utils")
COMP_TEST_COVER := atomic $(shell go list ./test/... | grep -v "utils")
component-test: common-component-test ## Run component tests

go-fuzz: FUZZ_FUNCS ?= FuzzGetVNC FuzzStartVM FuzzStopVM FuzzRestartVM FuzzDeletePod FuzzListAppEndpoints FuzzListAppWorkloads
go-fuzz: FUZZ_FUNC_PATH := ./internal/northbound/services/v2/resource
go-fuzz: common-go-fuzz-test ## Run go fuzz tests

#### Sub-targets ####

go-build: ## Build resource manager binaries
go-build: go-build-app-resource-manager go-build-vnc-proxy go-build-rest-proxy

go-build-app-resource-manager: ## Build the app-resource-manager binary
go-build-app-resource-manager: common-go-build-app-resource-manager

go-build-vnc-proxy: ## Build the vnc-proxy binary
go-build-vnc-proxy: common-go-build-vnc-proxy

go-build-rest-proxy: ## Build the rest-proxy binary
go-build-rest-proxy: common-go-build-rest-proxy

go-run: go-build ## Run the resource manager
	$(GOCMD) run $(GOEXTRAFLAGS) ./cmd/$(BINARY_NAME).go --initResourcesDefinitionPath ./configuration/default/resources.json

clean: common-clean clean-all ## Delete all build artifacts
	rm -rf node_modules package-lock.json

docker-build: docker-build-app-resource-manager docker-build-vnc-proxy docker-build-rest-proxy ## Build Docker image

docker-build-app-resource-manager: ## Build the app-resource-manager image
docker-build-app-resource-manager: DOCKER_NAME := $(ARM_DOCKER_NAME)
docker-build-app-resource-manager: DOCKER_BUILD_TARGET := app-resource-manager
docker-build-app-resource-manager: common-docker-build-app-resource-manager

docker-build-vnc-proxy: ## Build the vnc-proxy image
docker-build-vnc-proxy: DOCKER_NAME := $(VNC_PROXY_DOCKER_NAME)
docker-build-vnc-proxy: DOCKER_BUILD_TARGET := vnc-proxy
docker-build-vnc-proxy: common-docker-build-vnc-proxy

docker-build-rest-proxy: ## Build the rest-proxy image
docker-build-rest-proxy: DOCKER_NAME := $(REST_PROXY_DOCKER_NAME)
docker-build-rest-proxy: DOCKER_BUILD_TARGET := rest-proxy
docker-build-rest-proxy: common-docker-build-rest-proxy

docker-push: docker-push-app-resource-manager docker-push-vnc-proxy docker-push-rest-proxy ## Tag and push Docker image

docker-push-app-resource-manager: ## Push the app-resource-manager image
docker-push-app-resource-manager: DOCKER_BUILD_FLAGS += --push
docker-push-app-resource-manager: docker-build-app-resource-manager

docker-push-vnc-proxy: ## Push the vnc-proxy image
docker-push-vnc-proxy: DOCKER_BUILD_FLAGS += --push
docker-push-vnc-proxy: docker-build-vnc-proxy

docker-push-rest-proxy: ## Push the rest-proxy image
docker-push-rest-proxy: DOCKER_BUILD_FLAGS += --push
docker-push-rest-proxy: docker-build-rest-proxy

docker-list: docker-list-app-resource-manager docker-list-vnc-proxy docker-list-rest-proxy

docker-list-app-resource-manager: DOCKER_NAME := $(ARM_DOCKER_NAME)
docker-list-app-resource-manager: common-docker-list-app-resource-manager

docker-list-vnc-proxy: DOCKER_NAME := $(VNC_PROXY_DOCKER_NAME)
docker-list-vnc-proxy: common-docker-list-vnc-proxy

docker-list-rest-proxy: DOCKER_NAME := $(REST_PROXY_DOCKER_NAME)
docker-list-rest-proxy: common-docker-list-rest-proxy

helm-build-clean: common-helm-package-clean

helm-build: common-helm-package ## Package helm charts.

helm-push: common-helm-push ## Push helm charts.

rollup-rfb:
	@# Help: Rollup the NoVNC RFB Javascript files
	mkdir _build
	wget ${NO_VNC_RELEASES}/v${NO_VNC_VERSION}.tar.gz -O _build/noVNC-${NO_VNC_VERSION}.tar.gz
	tar -xvf _build/noVNC-${NO_VNC_VERSION}.tar.gz -C _build
	rm -rf _build/noVNC-${NO_VNC_VERSION}.tar.gz
	rollup --format es --input _build/noVNC-${NO_VNC_VERSION}/core/rfb.js -o vnc-proxy-web-login/rfb.js
	rm -rf _build/noVNC-${NO_VNC_VERSION}

lint-html:
	@# Help: Lint HTML files
	npx htmllint-cli vnc-proxy-web-ui/vnc-proxy-index.html

lint-css:
	@# Help: Lint CSS files
	@npm install stylelint stylelint-config-standard
	npx stylelint vnc-proxy-web-ui/vnc-proxy-styles.css

lint-js:
	@# Help: Lint JS files
	npx standard vnc-proxy-web-ui/vnc-proxy-main.js

## Coder targets for testing

kind-load: kind-load-app-resource-manager kind-load-vnc-proxy kind-load-rest-proxy

kind-load-app-resource-manager: DOCKER_NAME := $(ARM_DOCKER_NAME)
kind-load-app-resource-manager: DOCKER_REGISTRY := registry-rs.edgeorchestration.intel.com
kind-load-app-resource-manager: common-kind-load-app-resource-manager

kind-load-vnc-proxy: DOCKER_NAME := $(VNC_PROXY_DOCKER_NAME)
kind-load-vnc-proxy: DOCKER_REGISTRY := registry-rs.edgeorchestration.intel.com
kind-load-vnc-proxy: common-kind-load-vnc-proxy

kind-load-rest-proxy: DOCKER_NAME := $(REST_PROXY_DOCKER_NAME)
kind-load-rest-proxy: DOCKER_REGISTRY := registry-rs.edgeorchestration.intel.com
kind-load-rest-proxy: common-kind-load-rest-proxy


use-orch-context:
	kubectl config use-context ${MGMT_CLUSTER}

coder-clean:
	kubectl patch application -n dev root-app --type=merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false}}}}'
	kubectl delete application -n dev app-resource-manager --ignore-not-found=true

coder-upgrade-arm:
	helm upgrade --install -n ${ARM_NS} app-resource-manager -f $(CODER_DIR)/argocd/applications/configs/app-resource-manager.yaml  build/_output/$(ARM_HELM_CHART)
	helm -n ${ARM_NS} ls

coder-redeploy: helm-build use-orch-context coder-clean coder-upgrade-arm ## Redeploy app-resource-manager service

coder-rebuild: KIND_CLUSTER_NAME ?= $(MGMT_NAME)
coder-rebuild: coder-rebuild-arm ## Rebuild the app-resource-manager service

coder-rebuild-arm: ## Rebuild the app-resource-manager service from source and redeploy into Coder KinD cluster
coder-rebuild-arm: use-orch-context kind-load-app-resource-manager kind-load-vnc-proxy kind-load-rest-proxy
	kubectl -n ${ARM_NS} delete pod -l app=app-resource-manager
	kubectl -n ${ARM_NS} delete pod -l app=vnc-proxy-app-resource-manager


#### Unsupported targets ###

dependency-check-ci: ## Unsupported target
	echo "no dependency check"
