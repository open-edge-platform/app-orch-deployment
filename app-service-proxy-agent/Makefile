# Makefile for App Service Proxy

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
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
PROJECT_NAME    := app-service-proxy-agent
PROXY_AGENT     := proxy-agent
SERVICE_CONTROLLER := service-controller
TOKEN_REFRESHER := token-refresher
CHART_PATH      := deployments/app-service-proxy-agent

# Versioning variables
VERSION              := $(shell cat VERSION)
CHART_VERSION        := $(shell cat VERSION)
DOCKER_IMG_NAME      := app-service-proxy-agent
DOCKER_NAME          := $(DOCKER_IMG_NAME)
DOCKER_IMG_VERSION   ?= ${VERSION}
GIT_HASH_SHORT       := $(shell git rev-parse --short=8 HEAD)
OPA_IMAGE_VER = 0.67.1-static
GIT_COMMIT         ?= $(shell git rev-parse --short HEAD)
HELM_CHART_NAME	     ?= app-service-proxy-agent
VERSION_DEV_SUFFIX := -dev

# Add an identifying suffix for `-dev` builds only.
# Release build versions are verified as unique by the CI build process.
ifeq ($(patsubst %$(VERSION_DEV_SUFFIX),,$(lastword $(VERSION))),)
    DOCKER_VERSION ?= $(VERSION)-$(GIT_COMMIT)
else
    DOCKER_VERSION ?= $(VERSION)
endif

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
ENVTEST_K8S_VERSION = 1.27.1
ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)"
GO_TEST_DEPS ?= manifests generate fmt vet envtest
## Tool Versions
KUSTOMIZE_VERSION ?= v5.0.1
CONTROLLER_TOOLS_VERSION ?= v0.14.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)



.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest


YAML_FILES := $(shell find . -path './venv-env' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )

# Test variables
# Set TEST_TARGET to '<testname1>' or '<testname1\|testname2>' to run specific tests or use regex '<testname.*>' - example:
#  TEST_TARGET='TestAuth.*' make test
TEST_TARGET ?= .
# Default TEST_PKG, overwrite to select particular test pakages

TEST_PKG ?= $(shell go list ./... ) 

TEST_COVER ?= atomic $(shell go list ./... )

# Include shared makefile
include ../common.mk

# Security config for Go builds
GOEXTRAFLAGS += $(COMMON_GOEXTRAFLAGS)

#### Development Targets ####

.PHONY: build
build: go-build ## Build local binaries

## TODO: Re-enable the mdlint tool
## TODO: Re-enable the hadolint tool
.PHONY: lint
lint: license helmlint go-lint # hadolint mdlint ## Run license, helmlint, go-lint lint tools

test: go-test ## Run all unit tests

#### Sub-targets ####

go-build: ## Build resource manager binary
go-build: go-build-proxy-agent go-build-service-controller go-build-token-refresher

go-build-proxy-agent: ## Build the proxy-agent binary
go-build-proxy-agent: common-go-build-proxy-agent

go-build-service-controller: ## Build the service-controller binary
go-build-service-controller: common-go-build-service-controller

go-build-token-refresher: ## Build the token-refresher binary
go-build-token-refresher: common-go-build-token-refresher

go-run: go-build ## Run the resource manager
	$(GOCMD) run $(GOEXTRAFLAGS) ./cmd/$(BINARY_NAME).go --initResourcesDefinitionPath ./configuration/default/resources.json

clean: common-clean clean-all ## Delete all build artifacts

docker-build: docker-build-app-service-proxy-agent ## Build Docker image
docker-build-app-service-proxy-agent: DOCKER_NAME   := $(DOCKER_NAME)
docker-build-app-service-proxy-agent: DOCKER_BUILD_TARGET := app-service-proxy-agent
docker-build-app-service-proxy-agent: common-docker-build-app-service-proxy-agent

docker-push: docker-push-app-service-proxy-agent ## Tag and push Docker image
docker-push-app-service-proxy-agent: DOCKER_BUILD_FLAGS += --push
docker-push-app-service-proxy-agent: common-docker-build-app-service-proxy-agent

helm-build-clean: common-helm-package-clean

helm-build: common-helm-package ## Package helm charts.

helm-push: common-helm-push## Push helm charts.

#### Unsupported targets ###

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

dependency-check-ci: ## Unsupported target
	echo "no dependency check"
