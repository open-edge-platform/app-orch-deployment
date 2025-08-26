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
PROJECT_NAME    := app-deployment-manager
BINARY_NAME     := app-deployment-manager
REST_PROXY_NAME := rest-proxy
CHART_PATH      := deployment/charts/app-deployment-manager
TMP_DIR                     = /tmp

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
ADM_HELM_CHART_NAME	    ?= app-deployment-manager
ADM_CRD_HELM_CHART_NAME ?= app-deployment-crd
DOCKER_TAG              ?= ${VERSION}
DOCKER_REGISTRY         ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
DOCKER_REPOSITORY       ?= edge-orch
DOCKER_SUB_PROJ         ?= app
ADM_DOCKER_NAME         ?= adm-controller
ADM_GATEWAY_NAME        ?= adm-gateway
DOCKER_IMAGE            ?= ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${DOCKER_SUB_PRO}/${ADM_DOCKER_NAME}:${DOCKER_TAG}
GW_DOCKER_IMAGE         ?= ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${DOCKER_SUB_PRO}/${ADM_GATEWAY_NAME}:${DOCKER_TAG}
HELM_REGISTRY           ?= oci://080137407410.dkr.ecr.us-west-2.amazonaws.com
HELM_REPOSITORY         ?= edge-orch
HELM_SUB_PROJ           ?= app
HELM_CHART_PREFIX   	?= charts
HELM_DIRS = $(shell find ./deployment/charts -maxdepth 1 -mindepth 1 -type d -print )

# Coder variables
MGMT_NAME           ?= kind
MGMT_CLUSTER        ?= kind-${MGMT_NAME}
CODER_DIR           ?= ~/edge-manageability-framework
HELM_VERSION        ?= $(shell cat VERSION)
ADM_HELM_CHART      ?= app-deployment-manager-${HELM_VERSION}.tgz
ADM_CRD_HELM_CHART  ?= app-deployment-crd-${HELM_VERSION}.tgz
ADM_NS              ?= orch-app

# Tool Binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

# Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.16.3

# Test variables
# Set TEST_TARGET to '<testname1>' or '<testname1\|testname2>' to run specific tests or use regex '<testname.*>' - example:
#  TEST_TARGET='TestAuth.*' make test
TEST_TARGET ?= .
# Default TEST_PKG, overwrite to select particular test pakages
TEST_PKG ?= $(shell go list ./... | grep -v "internal" | tr '\n' ,)
TEST_COVER := atomic ./...

FUZZ_SECONDS ?= 60

EXCLUDE_PKGS_TEST := grep -v ./pkg/api | grep -v ./api | grep -v ./pkg/utils/ratelimiter | grep -v ./test \
	| grep -v ./internal/grafana/mockery | grep -v ./internal/catalogclient/mockery | grep -v ./internal/catalogclient/mocks \
	| grep -v ./pkg/fleet/mocks

PKG_LIST := ./cmd/... ./controllers/... ./webhooks/... ./pkg/... ./internal/...

ENVTEST ?= $(LOCALBIN)/setup-envtest
ENVTEST_K8S_VERSION = 1.25.0
OPA_FILE_PATH ?= deployment/charts/app-deployment-manager/files/openpolicyagent
OPA_IMAGE_VER       = 1.5.0-static
GOLANG_GOCOV_VERSION := latest
GOLANG_GOCOV_XML_VERSION := latest

# Include shared makefile
include ../common.mk

# Security config for Go builds
GOEXTRAFLAGS += $(COMMON_GOEXTRAFLAGS)

#### Buf protobuf code generation tooling ###

APIPKG_DIR ?= pkg/api

install-protoc-plugins: common-install-protoc-plugins ## Install protoc, buf and protobuf tools

verify-protoc-plugins: common-verify-protoc-plugins  ## Verify protoc plugins

OPENAPI_SPEC_FILE ?= ./api/nbi/v2/spec/openapi.yaml
REST_CLIENT_DIR ?= ./api/nbi/v2/pkg/restClient
proto-generate: common-buf-update common-buf-generate common-openapi-spec-validate common-rest-client-gen common-buf-lint ## Generate protobuf code

buf-lint-fix: common-buf-lint-fix ## Lint and format protobuf files

buf-update: common-buf-update ## Update buf modules

buf-lint: common-buf-lint ## Lint and format protobuf files

rest-client-gen: common-rest-client-gen ## Generate REST client code

#### Development Targets ####

controller-gen: ## Download controller-gen locally if necessary.
	test -s $(CONTROLLER_GEN) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

generate-crds: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./api/v1beta1/..." output:crd:artifacts:config=config/crd/bases

generate-apis: controller-gen
	$(CONTROLLER_GEN) object:headerFile="controllers/license.go.txt" paths="./api/v1beta1/..."

manifests: generate-apis generate-crds ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.

build: go-build ## Build local binaries

# FIXME: Fix the markdown linting errors and then re-enable the mdlint tool
# FIXME: Enable hadolint for Dockerfile linting (disabled due to false positive
# when COPYing --from a named build context
lint: license helmlint go-lint opa-lint rego-service-write-rule-match #hadolint #mdlint  ## Run license, helmlint, go-lint lint tools

docker-opa:
	docker pull openpolicyagent/opa:$(OPA_IMAGE_VER)

opa-lint: docker-opa
	docker run -v $(shell pwd)/${CHART_PATH}/files/openpolicyagent:/policies openpolicyagent/opa:$(OPA_IMAGE_VER) check  policies/

.PHONY: rego-service-write-rule-match
rego-service-write-rule-match: ## For every service request in Proto we expect a corresponding REGO rule
	@egrep -oh "\((Create|Update|Delete|List|Get).*Request" api/nbi/v2/deployment/v1/service.proto | awk -F'(' '{print $$2}' > ${TMP_DIR}/list_service_requests_out;
	@egrep -oh "\((Create|Update|Delete|List|Get).*Request" api/nbi/v2/deployment/v1/cluster_service.proto | awk -F'(' '{print $$2}' >> ${TMP_DIR}/list_service_requests_out;
	@egrep -oh "(Create|Update|Delete|List|Get).*Request if {" ${CHART_PATH}/files/openpolicyagent/*.rego | awk '{print $$1}' | sort > ${TMP_DIR}/list_rego_rules_out;
	sort ${TMP_DIR}/list_service_requests_out | diff - ${TMP_DIR}/list_rego_rules_out;

opa-test: docker-opa ## Test OPA policies
	docker run -v $(shell pwd)/$(OPA_FILE_PATH):/policies openpolicyagent/opa:$(OPA_IMAGE_VER) test --coverage policies -vv

envtest: $(ENVTEST) ## Download envtest-setup locally if necessary
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

test: manifests fmt vet envtest opa-test ## Run tests
	go install github.com/axw/gocov/gocov@${GOLANG_GOCOV_VERSION}
	go install github.com/AlekSi/gocov-xml@${GOLANG_GOCOV_XML_VERSION}
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		go test -gcflags="-l" -race -coverpkg=$$(go list ./... | ${EXCLUDE_PKGS_TEST} | tr '\n' ,) -coverprofile cover.out \
		-covermode atomic `go list ${PKG_LIST} | grep -v "/mocks" | grep -v "/mockery" | ${EXCLUDE_PKGS_TEST}`
	${GOBIN}/gocov convert cover.out | ${GOBIN}/gocov-xml > coverage.xml
	go tool cover -html=cover.out -o coverage.html

COMP_TEST_COVER := atomic ./test/...
PARALLEL_TESTS := 10
PARALLEL_SUITES := 10
component-test: common-component-test ## Run component tests

go-fuzz: FUZZ_FUNCS ?= FuzzCreateDeployment FuzzListDeployments$$ FuzzGetDeployment FuzzDeleteDeployment \
	FuzzUpdateDeployment FuzzDeploymentsStatus FuzzListDeploymentsPerCluster FuzzListDeploymentClusters \
	FuzzGetCluster FuzzListClusters FuzzGetKubeConfig FuzzGetAppNamespace FuzzRESTRouter

go-fuzz: FUZZ_FUNC_PATH := ./internal/northbound/fuzztests
go-fuzz: common-go-fuzz-test ## Run go fuzz tests

#### Sub-targets ####

go-build: ## Build deployment manager binaries
go-build: go-build-app-deployment-manager go-build-rest-proxy

go-build-app-deployment-manager: ## Build the app-deployment-manager binary
go-build-app-deployment-manager: common-go-build-app-deployment-manager
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(BIN_DIR)/manager main.go


go-build-rest-proxy: ## Build the rest-proxy binary
go-build-rest-proxy: common-go-build-rest-proxy

go-run: go-build ## Run the deployment manager
	$(GOCMD) run $(GOEXTRAFLAGS) ./cmd/$(BINARY_NAME).go

clean: common-clean clean-all ## Delete all build artifacts

docker-build: docker-build-app-deployment-manager docker-build-app-deployment-manager-gateway ## Build Docker image of app-deployment-manager Controller and Gateway

docker-build-app-deployment-manager: ## Build the app-deployment-manager controller image
docker-build-app-deployment-manager: DOCKER_NAME := $(ADM_DOCKER_NAME)
docker-build-app-deployment-manager: common-docker-build-app-deployment-manager

docker-build-app-deployment-manager-gateway: ## Build the app-deployment-manager gateway image
docker-build-app-deployment-manager-gateway: DOCKER_NAME := $(ADM_GATEWAY_NAME)
docker-build-app-deployment-manager-gateway: DOCKER_BUILD_FLAGS += -f build/Dockerfile.gateway
docker-build-app-deployment-manager-gateway: common-docker-build-app-deployment-manager-gateway

docker-push: docker-push-app-deployment-manager docker-push-app-deployment-manager-gateway

docker-list: docker-list-app-deployment-manager docker-list-app-deployment-manager-gateway

docker-list-app-deployment-manager: ## Build the app-deployment-manager controller image
docker-list-app-deployment-manager: DOCKER_NAME := $(ADM_DOCKER_NAME)
docker-list-app-deployment-manager: common-docker-list-app-deployment-manager

docker-list-app-deployment-manager-gateway: ## list the app-deployment-manager gateway image
docker-list-app-deployment-manager-gateway: DOCKER_NAME := $(ADM_GATEWAY_NAME)
docker-list-app-deployment-manager-gateway: common-docker-list-app-deployment-manager-gateway

## Tag and push Docker image

docker-push-setup:
	aws ecr create-repository --region us-west-2 --repository-name $(DOCKER_REPOSITORY)/$(DOCKER_SUB_PROJ)/$(DOCKER_NAME) || true

docker-push-app-deployment-manager: ## Push the app-deployment-manager image
docker-push-app-deployment-manager: DOCKER_NAME := $(ADM_DOCKER_NAME)
docker-push-app-deployment-manager: DOCKER_BUILD_FLAGS += --push
docker-push-app-deployment-manager: common-docker-push-app-deployment-manager

docker-push-app-deployment-manager-gateway: ## Push the app-deployment-manager image
docker-push-app-deployment-manager-gateway: DOCKER_NAME := $(ADM_GATEWAY_NAME)
docker-push-app-deployment-manager-gateway: DOCKER_BUILD_FLAGS += -f build/Dockerfile.gateway
docker-push-app-deployment-manager-gateway: DOCKER_BUILD_FLAGS += --push
docker-push-app-deployment-manager-gateway: common-docker-push-app-deployment-manager-gateway

helm-build-clean: HELM_DIRS = $(shell find ./deployment/charts -maxdepth 1 -mindepth 1 -type d -print )
helm-build-clean: common-helm-package-clean

helm-build: HELM_DIRS = $(shell find ./deployment/charts -maxdepth 1 -mindepth 1 -type d -print )
helm-build: common-helm-package ## Package helm charts.

helm-package: helm-build

helm-push: helm-push-adm helm-push-crd

helm-push-adm: HELM_CHART_NAME := $(ADM_HELM_CHART_NAME)
helm-push-adm:  common-helm-push-adm ## Push helm charts.

helm-push-crd: HELM_CHART_NAME := $(ADM_CRD_HELM_CHART_NAME)
helm-push-crd:  common-helm-push-crd ## Push helm charts.


#### Coder targets for testing

kind-load: kind-load-app-deployment-manager kind-load-app-deployment-manager-gateway

kind-load-app-deployment-manager: DOCKER_NAME := $(ADM_DOCKER_NAME)
kind-load-app-deployment-manager: DOCKER_REGISTRY := registry-rs.edgeorchestration.intel.com
kind-load-app-deployment-manager: common-kind-load-app-deployment-manager

kind-load-app-deployment-manager-gateway: DOCKER_NAME := $(ADM_GATEWAY_NAME)
kind-load-app-deployment-manager-gateway: DOCKER_REGISTRY := registry-rs.edgeorchestration.intel.com
kind-load-app-deployment-manager-gateway: DOCKER_BUILD_FLAGS += -f build/Dockerfile.gateway
kind-load-app-deployment-manager-gateway: common-kind-load-app-deployment-manager-gateway

use-orch-context:
	kubectl config use-context ${MGMT_CLUSTER}

coder-clean:
	kubectl patch application -n dev root-app --type=merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false}}}}'
	kubectl delete application -n dev app-deployment-manager --ignore-not-found=true

coder-upgrade-adm:
	helm upgrade --install -n ${ADM_NS} ma-adm -f $(CODER_DIR)/argocd/applications/configs/app-deployment-manager.yaml  build/_output/$(ADM_HELM_CHART)
	helm -n ${ADM_NS} ls

coder-redeploy: helm-package use-orch-context coder-clean coder-upgrade-adm ## Redeploy app-deployment-manager Controller and Gateway in the Coder KinD cluster
coder-redeploy-adm-crd: helm-package use-orch-context coder-clean coder-clean-adm-crd coder-upgrade-adm-crd ## Redeploy app-deployment-manager Controller, Gateway and the CRDs in the Coder KinD cluster

coder-clean-adm-crd:
	kubectl patch application -n dev root-app --type=merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false}}}}'
	kubectl delete application -n dev app-deployment-manager --ignore-not-found=true
	kubectl delete application -n dev app-deployment-crd --ignore-not-found=true
	kubectl delete crd clusters.app.edge-orchestrator.intel.com --ignore-not-found=true
	kubectl delete crd deploymentclusters.app.edge-orchestrator.intel.com --ignore-not-found=true
	kubectl delete crd deployments.app.edge-orchestrator.intel.com --ignore-not-found=true

coder-upgrade-adm-crd:
	helm upgrade --install -n ${ADM_NS} ma-adm-crd -f $(CODER_DIR)/argocd/applications/configs/app-deployment-manager.yaml  build/_output/$(ADM_CRD_HELM_CHART)
	helm upgrade --install -n ${ADM_NS} ma-adm -f $(CODER_DIR)/argocd/applications/configs/app-deployment-manager.yaml  build/_output/$(ADM_HELM_CHART)
	helm -n ${ADM_NS} ls

coder-rebuild: KIND_CLUSTER_NAME ?= $(MGMT_NAME)
coder-rebuild: coder-rebuild-adm coder-rebuild-gateway ## Rebuild the app-deployment-manager Controller and Gateway images and load into Coder KinD cluster

coder-rebuild-adm: ## Rebuild the app-deployment-manager Controller from source and redeploy into Coder KinD cluster
coder-rebuild-adm: use-orch-context kind-load-app-deployment-manager
	kubectl -n ${ADM_NS} delete pod -l app=app-deployment-manager

coder-rebuild-gateway: ## Rebuild the app-deployment-manager Gateway from source and redeploy into Coder KinD cluster
coder-rebuild-gateway: use-orch-context kind-load-app-deployment-manager-gateway
	kubectl -n ${ADM_NS} delete pod -l app=app-deployment-api


#### Unsupported targets ###

dependency-check-ci: ## Unsupported target
	echo "no dependency check"
