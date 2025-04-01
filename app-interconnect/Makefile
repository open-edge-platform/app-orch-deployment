# Makefile for Interconnect

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
PROJECT_NAME    := app-interconnect
BINARY_NAME     := app-interconnect-manager
CHART_PATH      := deploy/charts/app-interconnect-manager

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

DOCKER_NAME       ?= app-interconnect-manager
DOCKER_TAG        ?= ${VERSION}
DOCKER_REGISTRY   ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
DOCKER_REPOSITORY ?= edge-orch
DOCKER_SUB_PROJ   ?= app
DOCKER_IMAGE      ?= ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${DOCKER_SUB_PRO}/${DOCKER_NAME}:${DOCKER_TAG}

HELM_REGISTRY     ?= oci://080137407410.dkr.ecr.us-west-2.amazonaws.com
HELM_REPOSITORY   ?= edge-orch
HELM_SUB_PROJ     ?= app
HELM_CHART_PREFIX ?= charts
HELM_CHART_NAME	  ?= app-interconnect-manager
HELM_CHART_PATH   := deploy/charts/app-interconnect-manager
HELM_DIRS         := $(shell find ./deploy/charts -maxdepth 1 -mindepth 1 -type d -print )



# Test variables
# Set TEST_TARGET to '<testname1>' or '<testname1\|testname2>' to run specific tests or use regex '<testname.*>' - example:
#  TEST_TARGET='TestAuth.*' make test
TEST_TARGET ?= .
# Default TEST_PKG, overwrite to select particular test pakages
TEST_PKG ?= $(shell go list ./... | grep -v "internal" | tr '\n' ,)
TEST_COVER := atomic ./...

# Include shared makefile
include ../common.mk

# Security config for Go builds
GOEXTRAFLAGS += $(COMMON_GOEXTRAFLAGS)

#### Buf protobuf code generation tooling ###

APIPKG_DIR ?= pkg/api

buf-generate: $(VENV_NAME) ## Compile protobuf files in api into code
	set +u; . ./$</bin/activate; set -u ;\
        buf --version ;\
        buf generate

buf-lint-fix: $(VENV_NAME) ## Lint and when possible fix protobuf files
	buf --version
	buf format -d -w
	buf lint

buf-update: common-buf-update ## Update buf modules

buf-lint: common-buf-lint ## Lint and format protobuf files

#### Development Targets ####

build: go-build ## Build local binaries

# FIXME: Fix the markdown linting errors and then re-enable the mdlint tool
# FIXME: Enable hadolint for Dockerfile linting (disabled due to false positive
# when COPYing --from a named build context
lint: license helmlint go-lint #hadolint #mdlint ## Run license, helmlint, go-lint lint tools

EXCLUDE_PKGS_TEST := grep -v $(PKG)/pkg/api | grep -v $(PKG)/api | grep -v $(PKG)/pkg/utils/ratelimiter | grep -v $(PKG)/internal/ingresshandler \
	| grep -v $(PKG)/internal/grafana/mockery | grep -v $(PKG)/internal/catalogclient/mockery | grep -v $(PKG)/internal/catalogclient/mocks

test:

#### Sub-targets ####

go-build: ## Build deployment manager binaries
go-build: common-go-build-interconnect-manager

go-run: go-build ## Run the interconnect manager.
	$(GOCMD) run $(GOEXTRAFLAGS) ./cmd/$(BINARY_NAME).go

clean: clean-all common-clean ## Delete all build artifacts

## Tag and push Docker image

docker-build: ## Build the app-interconnect-manager image
docker-build: DOCKER_BUILD_FLAGS += -f build/Dockerfile
docker-build: common-docker-build

docker-push-setup:
	aws ecr create-repository --region us-west-2 --repository-name $(DOCKER_REPOSITORY)/$(DOCKER_SUB_PROJ)/$(DOCKER_NAME) || true

docker-push: ## Push the app-interconnect-manager image
docker-push: DOCKER_BUILD_FLAGS += -f build/Dockerfile
docker-push: common-docker-push

helm-build-clean: HELM_DIRS = $(shell find ./deploy/charts -maxdepth 1 -mindepth 1 -type d -print )
helm-build-clean: common-helm-package-clean

helm-build: HELM_DIRS = $(shell find ./deploy/charts -maxdepth 1 -mindepth 1 -type d -print )
helm-build: common-helm-package ## Package helm charts.

helm-push: ## Push helm charts.
helm-push: common-helm-push


#### Unsupported targets ###

dependency-check-ci: ## Unsupported target
	echo "no dependency check"
