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
PROJECT_NAME    := app-service-proxy
BINARY_NAME     := app-service-proxy
CHART_PATH      := deployments/app-service-proxy

# Versioning variables
VERSION              := $(shell cat VERSION)
CHART_VERSION        := $(shell cat VERSION)
VERSION_DEV_SUFFIX   := -dev
DOCKER_IMG_NAME      := app-service-proxy
DOCKER_NAME          := $(DOCKER_IMG_NAME)
ASP_DOCKER_NAME      := $(DOCKER_IMG_NAME)
DOCKER_IMG_VERSION   ?= ${VERSION}
OPA_IMAGE_VER        = 1.10.1-static
GIT_COMMIT           ?= $(shell git rev-parse --short HEAD)
HELM_CHART_NAME	     ?= app-service-proxy
HELM_CHART_PATH	     ?= ./deployments/${HELM_CHART_NAME}
HELM_DIRS            ?= $(HELM_CHART_PATH)
OPA_FILE_PATH        ?= $(HELM_CHART_PATH)/files/openpolicyagent

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

## Tool Versions
KUSTOMIZE_VERSION ?= v5.0.1
CONTROLLER_TOOLS_VERSION ?= v0.12.0

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


# Yamllint variables
YAML_FILES         := $(shell find . -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
YAML_IGNORE        := vendor, deployments, $(VENV_NAME)

# Test variables
# Set TEST_TARGET to '<testname1>' or '<testname1\|testname2>' to run specific tests or use regex '<testname.*>' - example:
#  TEST_TARGET='TestAuth.*' make test
TEST_TARGET ?= .
# Default TEST_PKG, overwrite to select particular test pakages

TEST_PKG ?= $(shell go list ./... | grep -v "internal/admclient" | grep -v "internal/auth" | grep -v "internal/rbac" | grep -v "internal/vault" | grep -v "cmd" | grep -v "controller" | tr '\n' ,)

TEST_COVER ?= atomic $(shell go list ./cmd/... ./internal/... | grep -v "internal/admclient" | grep -v "internal/auth" | grep -v "internal/rbac" | grep -v "internal/vault" | grep -v "cmd" | grep -v "controller")

FUZZ_SECONDS ?= 60

# Include shared makefile
include ../common.mk

# Security config for Go builds
GOEXTRAFLAGS += $(COMMON_GOEXTRAFLAGS)

#### Development Targets ####

build: go-build ## Build local binaries

lint: helmlint go-lint yamllint lint-html lint-css lint-js opa-lint hadolint mdlint ## Run helmlint, go-lint, yamllint lint tools

lint-html:
	@# Help: Lint HTML files
	npx htmllint-cli web-login/app-service-proxy-index.html

lint-css:
	@# Help: Lint CSS files
	@npm install stylelint stylelint-config-standard
	npx stylelint web-login/app-service-proxy-styles.css

lint-js:
	@# Help: Lint JS files
	npx standard web-login/app-service-proxy-main.js

test: go-test opa-test ## Run all unit tests

docker-opa:
	docker pull openpolicyagent/opa:$(OPA_IMAGE_VER)

opa-lint: docker-opa
	docker run -v $(shell pwd)/${CHART_PATH}/files/openpolicyagent:/policies openpolicyagent/opa:$(OPA_IMAGE_VER) check  policies/

opa-test: docker-opa ## Test OPA policies
	docker run -v $(shell pwd)/$(OPA_FILE_PATH):/policies openpolicyagent/opa:$(OPA_IMAGE_VER) test --coverage policies -vv

COMP_TEST_COVER := atomic $(shell go list ./test/... )
component-test: common-component-test ## Run component tests

go-fuzz: FUZZ_FUNCS ?= FuzzRouter FuzzProxyHeaderHost FuzzProxyHeaderProto
go-fuzz: FUZZ_FUNC_PATH := ./internal/server/fuzztests
go-fuzz: common-go-fuzz-test ## Run go fuzz tests

#### Sub-targets ####

go-build: ## Build resource manager binary
go-build: go-build-app-service-proxy

go-build-app-service-proxy: ## Build the app-service-proxy binary
go-build-app-service-proxy: common-go-build-app-service-proxy

go-run: go-build ## Run the resource manager
	$(GOCMD) run $(GOEXTRAFLAGS) ./cmd/$(BINARY_NAME).go --initResourcesDefinitionPath ./configuration/default/resources.json

clean: common-clean clean-all ## Delete all build artifacts
	rm -rf node_modules package-lock.json

docker-build: docker-build-app-service-proxy ## Build Docker image
docker-build-app-service-proxy: DOCKER_NAME   := $(DOCKER_NAME)
docker-build-app-service-proxy: DOCKER_BUILD_TARGET := app-service-proxy
docker-build-app-service-proxy: common-docker-build-app-service-proxy

docker-push: docker-push-app-service-proxy ## Tag and push Docker image
docker-push-app-service-proxy: DOCKER_BUILD_FLAGS += --push
docker-push-app-service-proxy: common-docker-build-app-service-proxy

docker-list: docker-list-app-service-proxy ## Build Docker image
docker-list-app-service-proxy: DOCKER_NAME   := $(DOCKER_NAME)
docker-list-app-service-proxy:  common-docker-list-app-service-proxy

helm-build-clean: common-helm-package-clean

helm-build: common-helm-package ## Package helm charts.

helm-push: common-helm-push## Push helm charts.

##@ Coder

.PHONY: coder-rebuild
kind-load: kind-load-app-service-proxy

kind-load-app-service-proxy: DOCKER_NAME := $(ASP_DOCKER_NAME)
kind-load-app-service-proxy: DOCKER_REGISTRY := registry-rs.edgeorchestration.intel.com
kind-load-app-service-proxy: common-kind-load-app-service-proxy

MAKEDIR          := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
API_PROXY_HELM_PKG  ?= $(MAKEDIR)/build/_output/app-service-proxy-${VERSION}.tgz
CODER_DIR ?= ~/edge-manageability-framework

use-orch-context:
	kubectl config use-context ${MGMT_CLUSTER}

coder-rebuild: kind-load-app-service-proxy ## Rebuild API Proxy for Coder environment test.
	kubectl -n orch-app delete pod -l app=app-service-proxy

.PHONY: coder-redeploy
coder-redeploy: helm-build ## Redeploy API Proxy for Coder environment test.
	kubectl config use-context kind-kind
	kubectl patch application -n dev root-app --type=merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false}}}}'
	kubectl delete application -n dev app-service-proxy --ignore-not-found=true
	helm upgrade --install -n orch-app ma-app-service-proxy -f $(CODER_DIR)/argocd/applications/configs/app-service-proxy.yaml $(API_PROXY_HELM_PKG)
	helm -n orch-app ls
#### Unsupported targets ###

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

dependency-check-ci: ## Unsupported target
	echo "no dependency check"
