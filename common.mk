# common.mk - common targets for App Orchestration modules

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Makefile Style Guide:
# - Help will be generated from ## comments at end of any target line
# - Use smooth parens $() for variables over curly brackets ${} for consistency
# - Continuation lines (after an \ on previous line) should start with spaces
#   not tabs - this will cause editor highligting to point out editing mistakes
# - When creating targets that run a lint or similar testing tool, print the
#   tool version first so that issues with versions in CI or other remote
#   environments can be caught

# Optionally include tool version checks, not used in Docker builds
ifeq ($(TOOL_VERSION_CHECK), 1)
	include ../version.mk
endif

#### Variables ####

## Shell config variables ##
SHELL := bash -eu -o pipefail

## GO variables ##
GOARCH	:= $(shell go env GOARCH)
GOCMD   := go
GOTESTSUM_PKG := gotest.tools/gotestsum@v1.12.2
OAPI_CODEGEN_VERSION ?= v2.2.0
LOCALBIN ?= $(shell pwd)/bin
BUF_VERSION ?= v1.57.0

## Path variables ##
OUT_DIR	:= out
BIN_DIR := bin

## Docker variables ##
DOCKER_ENV              := DOCKER_BUILDKIT=1
DOCKER_REGISTRY         ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
DOCKER_REPOSITORY       ?= edge-orch
DOCKER_SUB_PROJ  		?= app
DOCKER_LABEL_REPO_URL   ?= $(shell git remote get-url $(shell git remote | head -n 1))
DOCKER_LABEL_VERSION    ?= $(DOCKER_IMG_VERSION)
DOCKER_LABEL_REVISION   ?= $(GIT_COMMIT)
DOCKER_LABEL_BUILD_DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")
DOCKER_BUILD_FLAGS      :=

HELM_REGISTRY           ?= oci://080137407410.dkr.ecr.us-west-2.amazonaws.com
HELM_REPOSITORY         ?= edge-orch
HELM_SUB_PROJ           ?= app
HELM_CHART_PREFIX       ?= charts
HELM_CHART_BUILD_DIR    ?= build/_output/
HELM_CHART_PATH         ?= "./deployment/${HELM_CHART_NAME}"
HELM_DIRS               ?= $(shell find ./deployment/charts -maxdepth 1 -mindepth 1 -type d -print )

## Kind variables ##
KIND_CLUSTER_NAME := kind

# Security config for Go Builds - see:
#   https://readthedocs.intel.com/SecureCodingStandards/latest/compiler/golang/
# -trimpath: Remove all file system paths from the resulting executable.
# -gcflags="all=-m": Print optimizations applied by the compiler for review and verification against security requirements.
# -gcflags="all=-spectre=all" Enable all available Spectre mitigations
# -ldflags="all=-s -w" remove the symbol and debug info
# -ldflags="all=-X ..." Embed binary build stamping information
ifeq ($(GOARCH),arm64)
	# Note that arm64 (Apple, similar) does not support any spectre mititations.
  COMMON_GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre= -N -l" -asmflags="all=-spectre=" -ldflags="all=-s -w -X 'main.RepoURL=$(DOCKER_LABEL_REPO_URL)' -X 'main.Version=$(DOCKER_LABEL_VERSION)' -X 'main.Revision=$(DOCKER_LABEL_REVISION)' -X 'main.BuildDate=$(DOCKER_LABEL_BUILD_DATE)'"
else
  COMMON_GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -X 'main.RepoURL=$(DOCKER_LABEL_REPO_URL)' -X 'main.Version=$(DOCKER_LABEL_VERSION)' -X 'main.Revision=$(DOCKER_LABEL_REVISION)' -X 'main.BuildDate=$(DOCKER_LABEL_BUILD_DATE)'"
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

#### Directory Targets ####

$(OUT_DIR): ## Create out directory
	mkdir -p $(OUT_DIR)

$(BIN_DIR): ## Create bin directory
	mkdir -p $(BIN_DIR)


#### Build Targets ####

vendor:
	go mod vendor

mod-update: ## Update Go modules.
	$(GOCMD) mod tidy

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

# Define the target for installing all plugins
common-install-protoc-plugins:
	@echo "Installing protoc-gen-doc..."
	@go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest
	@echo "Installing protoc-gen-validate..."
	@go install github.com/envoyproxy/protoc-gen-validate@latest
	# @echo "Installing protoc-gen-go-grpc..."
	# @go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installing protoc-gen-connect-go..."
	@go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
	# @echo "Installing protoc-gen-connect-openapi..."
	# @go install github.com/bufbuild/protoc-gen-connect-openapi@latest
	# @echo "Installing official protoc-gen-openapi from Google..."
	# @go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	@echo "Installing oapi-codegen"
	@go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION}
	@echo "Installing buf"
	@go install github.com/bufbuild/buf/cmd/buf@${BUF_VERSION}
	@echo "*** You need to add "$(GOBIN)" directory to your PATH..."
	@echo
	@echo "All plugins installed successfully."

# Define a target to verify the installation of all plugins
common-verify-protoc-plugins:
	@echo "Verifying protoc-gen-doc installation..."
	@command -v protoc-gen-doc >/dev/null 2>&1 && echo "protoc-gen-doc is installed." || echo "---> protoc-gen-doc is not installed."
	@echo "Verifying protoc-gen-validate installation..."
	@command -v protoc-gen-validate >/dev/null 2>&1 && echo "protoc-gen-validate is installed." || echo "---> protoc-gen-validate is not installed."
	# @echo "Verifying protoc-gen-go-grpc installation..."
	# @command -v protoc-gen-go-grpc >/dev/null 2>&1 && echo "protoc-gen-go-grpc is installed." || echo "---> protoc-gen-go-grpc is not installed."
	@echo "Verifying protoc-gen-connect-go installation..."
	@command -v protoc-gen-connect-go >/dev/null 2>&1 && echo "protoc-gen-connect-go is installed." || echo "---> protoc-gen-connect-go is not installed."
	@echo "Verifying protoc-gen-connect-openapi installation..."
	@command -v protoc-gen-connect-openapi >/dev/null 2>&1 && echo "protoc-gen-connect-openapi is installed." || echo "---> protoc-gen-connect-openapi is not installed."
	# @echo "Verifying protoc-gen-openapi installation..."
	# @command -v protoc-gen-openapi >/dev/null 2>&1 && echo "protoc-gen-openapi is installed." || echo "---> protoc-gen-openapi is not installed."
	@echo "Verifying oapi-codegen installation..."
	@command -v oapi-codegen >/dev/null 2>&1 && echo "oapi-codegen is installed." || echo "---> oapi-codegen is not installed."
	@echo "Verifying buf installation..."
	@command -v buf >/dev/null 2>&1 && echo "buf is installed." || echo "---> buf is not installed."


#### Docker Targets ####

common-docker-setup-env:
ifdef DOCKER_BUILD_PLATFORM
	-docker buildx rm builder
	docker buildx create --name builder --use
endif

common-docker-build: ## Build a generic Docker image
common-docker-build: common-docker-build-generic

common-docker-build-%: ## Build Docker image
common-docker-build-%: DOCKER_BUILD_FLAGS   += $(if $(DOCKER_BUILD_PLATFORM),--load,)
common-docker-build-%: DOCKER_BUILD_FLAGS   += $(addprefix --platform ,$(DOCKER_BUILD_PLATFORM))
common-docker-build-%: DOCKER_BUILD_FLAGS   += $(addprefix --target ,$(DOCKER_BUILD_TARGET))
common-docker-build-%: DOCKER_VERSION       ?= latest
common-docker-build-%: DOCKER_LABEL_VERSION ?= $(DOCKER_VERSION)
common-docker-build-%: common-docker-setup-env
	$(if $(DOCKER_NAME),,$(error DOCKER_NAME is not set!))
	$(GOCMD) mod vendor
	aws ecr create-repository --region us-west-2 --repository-name $(DOCKER_REPOSITORY)/$(DOCKER_SUB_PROJ)/$(DOCKER_NAME) || true
	docker buildx build \
		$(DOCKER_BUILD_FLAGS) \
		-t $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_SUB_PROJ)/$(DOCKER_NAME):$(DOCKER_VERSION) \
		--build-context root=.. \
		--build-arg http_proxy="$(http_proxy)" --build-arg HTTP_PROXY="$(HTTP_PROXY)" \
		--build-arg https_proxy="$(https_proxy)" --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
		--build-arg no_proxy="$(no_proxy)" --build-arg NO_PROXY="$(NO_PROXY)" \
		--build-arg REPO_URL="$(DOCKER_LABEL_REPO_URL)" \
		--build-arg VERSION="$(DOCKER_LABEL_VERSION)" \
		--build-arg REVISION="$(DOCKER_LABEL_REVISION)" \
		--build-arg BUILD_DATE="$(DOCKER_LABEL_BUILD_DATE)" \
		.
	@rm -rf vendor

common-docker-load: ## Tag and load a generic Docker image
common-docker-load: common-docker-load-generic

common-docker-load-%: ## Tag and load Docker image
common-docker-load-%: DOCKER_BUILD_FLAGS += --load
common-docker-load-%: common-docker-build-%

common-docker-push: ## Tag and push a generic Docker image
common-docker-push: common-docker-push-generic

common-docker-push-%: ## Tag and push Docker image
common-docker-push-%: DOCKER_BUILD_FLAGS += --push
common-docker-push-%: common-docker-build-%
	echo "Pushing $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_SUB_PROJ)/$(DOCKER_NAME):$(DOCKER_VERSION)"

common-docker-list-%: ## Print name of docker container image
	@echo "  $(DOCKER_NAME):"
	@echo "    name: '$(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_SUB_PROJ)/$(DOCKER_NAME):$(DOCKER_VERSION)'"
	@echo "    version: '$(DOCKER_VERSION)'"
	@echo "    gitTagPrefix: '$(PROJECT_NAME)/v'"
	@echo "    buildTarget: '$(PROJECT_NAME)-docker-build'"

#### Go Targets ####

common-go-build: fmt common-go-build-generic

common-go-build-%: fmt $(BIN_DIR) ## Build resource manager binary
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(BIN_DIR)/$* ./cmd/$*


#### Python Targets ####

VENV_NAME	:= venv_$(PROJECT_NAME)

$(VENV_NAME): requirements.txt ## Create Python venv
	python3 -m venv $@ ;\
  set +u; . ./$@/bin/activate; set -u ;\
  python -m pip install --upgrade pip ;\
  python -m pip install openapi-spec-validator;\
  python -m pip install -r requirements.txt


#### Maintenance Targets ####

go-tidy: ## Run go mod tidy
	$(GOCMD) mod tidy

go-lint-fix: ## Apply automated lint/formatting fixes to go files
	golangci-lint run --fix --config .golangci.yml


#### Linting Targets ####

# https://github.com/koalaman/shellcheck
SH_FILES := $(shell find . -type f \( -name '*.sh' \) -print )
shellcheck: ## lint shell scripts with shellcheck
	shellcheck --version
	shellcheck -x -S style $(SH_FILES)

# https://pypi.org/project/reuse/
license: $(VENV_NAME) ## Check licensing with the reuse tool
	set +u; . ./$</bin/activate; set -u ;\
  reuse --version ;\
  reuse --root . lint

hadolint: ## Check Dockerfile with Hadolint
	hadolint Dockerfile

checksec: go-build ## Check various security properties that are available for executable,like RELRO, STACK CANARY, NX,PIE etc
	$(GOCMD) version -m $(OUT_DIR)/$(BINARY_NAME)
	checksec --output=json --file=$(OUT_DIR)/$(BINARY_NAME)
	checksec --fortify-file=$(OUT_DIR)/$(BINARY_NAME)

yamllint: $(VENV_NAME) ## Lint YAML files
#	. ./$</bin/activate; set -u ;\
#  yamllint --version ;\
#  yamllint -d '{extends: default, rules: {line-length: {max: 200}, braces: {min-spaces-inside: 0, max-spaces-inside: 5}, brackets: {min-spaces-inside: 0, max-spaces-inside: 5},colons: {max-spaces-before: 1, max-spaces-after: 5}}, ignore: [$(YAML_IGNORE)]}' -s $(YAML_FILES)
	@echo "YAML linting is currently disabled"

mdlint: ## Link MD files
	markdownlint --version ;\
	markdownlint "**/*.md" -c ../.markdownlint.yml

helmlint: ## Lint Helm charts.
	helm lint ${CHART_PATH}

GOLANG_CLI_LINT_VERSION := v1.64.8
go-lint: $(OUT_DIR) ## Run go lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOBIN} ${GOLANG_CLI_LINT_VERSION}
	${GOBIN}/golangci-lint --version
	${GOBIN}/golangci-lint -v --timeout 10m run $(LINT_DIRS) --config .golangci.yml

go-test: $(OUT_DIR) $(GO_TEST_DEPS) ## Run go test and calculate code coverage
	KUBEBUILDER_ASSETS=$(ASSETS) \
	$(GOCMD) test -race -v -p 1 \
	-coverpkg=$(TEST_PKG) -run $(TEST_TARGET) \
	-coverprofile=$(OUT_DIR)/coverage.out \
	-covermode $(TEST_COVER) $(if $(TEST_ARGS),-args $(TEST_ARGS)) \
	| tee >(go-junit-report -set-exit-code > $(OUT_DIR)/report.xml)
	gocover-cobertura $(if $(TEST_IGNORE_FILES),-ignore-files $(TEST_IGNORE_FILES)) < $(OUT_DIR)/coverage.out > $(OUT_DIR)/coverage.xml
	$(GOCMD) tool cover -html=$(OUT_DIR)/coverage.out -o $(OUT_DIR)/coverage.html
	$(GOCMD) tool cover -func=$(OUT_DIR)/coverage.out -o $(OUT_DIR)/function_coverage.log

PARALLEL_SUITES ?= 1
PARALLEL_TESTS ?= 1
common-component-test: ## Run component tests
	go run $(GOTESTSUM_PKG) --format=standard-verbose --jsonfile=test-report.json -- -p $(PARALLEL_SUITES) -parallel $(PARALLEL_TESTS) -timeout 30m -count=1 \
	-covermode $(COMP_TEST_COVER)

common-go-fuzz-test: ## GO fuzz tests
	for func in $(FUZZ_FUNCS); do \
		$(GOCMD) test $(FUZZ_FUNC_PATH) -fuzz $$func -fuzztime=${FUZZ_SECONDS}s -v; \
	done

#### Protobuf Targets ####

common-buf-lint-fix: $(VENV_NAME) ## Lint and when possible fix protobuf files
	buf --version
	buf format -d -w
	buf lint

common-buf-generate: $(VENV_NAME) ## Compile protobuf files in api into code
	set +u; . ./$</bin/activate; set -u ;\
        buf --version ;\
        buf generate

common-openapi-spec-validate: $(VENV_NAME)
		set +u; . ./$</bin/activate; set -u ;\
	openapi-spec-validator $(OPENAPI_SPEC_FILE)

common-buf-update: $(VENV_NAME) ## Update buf modules
	set +u; . ./$</bin/activate; set -u ;\
  buf --version ;\
  pushd api; buf mod update; popd ;\
  buf build

common-buf-lint: $(VENV_NAME) ## Lint and format protobuf files
	buf --version
	buf format -d --exit-code
	buf lint

#### Rest Client Targets ####
common-rest-client-gen: ## Generate rest-client.
	@echo Generate Rest client from the generated openapi spec.
	mkdir -p $(REST_CLIENT_DIR)
	oapi-codegen -generate client -old-config-style -package restClient -o $(REST_CLIENT_DIR)/client.go $(OPENAPI_SPEC_FILE)
	oapi-codegen -generate types -old-config-style -package restClient -o $(REST_CLIENT_DIR)/types.go $(OPENAPI_SPEC_FILE)

#### Helm Targets ####

common-helm-package-clean: ## Clean helm charts.
	for d in $(HELM_DIRS); do \
		yq eval -i '.version = "0.0.0"' $$d/Chart.yaml; \
		yq eval -i 'del(.appVersion)' $$d/Chart.yaml; \
		yq eval -i 'del(.annotations.revision)' $$d/Chart.yaml; \
		yq eval -i 'del(.annotations.created)' $$d/Chart.yaml; \
	done
	rm -f $(HELM_PKGS)

common-helm-package: ## Package helm charts.
	for d in $(HELM_DIRS); do \
		yq eval -i '.version = "${CHART_VERSION}"' $$d/Chart.yaml; \
		yq eval -i '.appVersion = "${DOCKER_VERSION}"' $$d/Chart.yaml; \
		yq eval -i '.annotations.revision = "${DOCKER_LABEL_REVISION}"' $$d/Chart.yaml; \
		yq eval -i '.annotations.created = "${DOCKER_LABEL_BUILD_DATE}"' $$d/Chart.yaml; \
		helm package --app-version=${DOCKER_VERSION} --version=${CHART_VERSION} --dependency-update  -u $$d --destination ${HELM_CHART_BUILD_DIR}; \
	done

common-helm-push: common-helm-push-generic

common-helm-push-%: ## Tag and push Docker image
	aws ecr create-repository --region us-west-2 --repository-name $(HELM_REPOSITORY)/$(HELM_SUB_PROJ)/$(HELM_CHART_PREFIX)/$(HELM_CHART_NAME) || true
	helm push ${HELM_CHART_BUILD_DIR}${HELM_CHART_NAME}-[0-9]*.tgz $(HELM_REGISTRY)/$(HELM_REPOSITORY)/$(HELM_SUB_PROJ)/$(HELM_CHART_PREFIX)

helm-list:
	@for d in $(HELM_DIRS); do \
    cname=$$(grep "^name:" "$$d/Chart.yaml" | cut -d " " -f 2) ;\
    echo "  $$cname:" ;\
    echo -n "    "; grep "^version" "$$d/Chart.yaml"  ;\
    echo "    gitTagPrefix: '${PROJECT_NAME}/v'" ;\
    echo "    outDir: '${PROJECT_NAME}/${HELM_CHART_BUILD_DIR}'" ;\
  done

#### Clean Targets ####

common-clean: ## Delete build and vendor directories
	rm -rf $(OUT_DIR) $(BIN_DIR) vendor
	go clean -testcache

clean-venv: ## Delete Python venv
	rm -rf "$(VENV_NAME)"

clean-all: clean-venv ## Delete all built artifacts and downloaded tools


#### Kind targets ####

common-kind-setup: ## Set up the KinD cluster
common-kind-setup:
	kind create cluster -n $(KIND_CLUSTER_NAME)

common-kind-load: ## Build the Docker image and load it into kind
common-kind-load: common-kind-load-generic

common-kind-load-%: ## Build the Docker image and load it into kind
common-kind-load-%: DOCKER_BUILD_FLAGS += --load
common-kind-load-%: common-docker-build-%
	kind load docker-image -n $(KIND_CLUSTER_NAME) $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_SUB_PROJ)/$(DOCKER_NAME):$(DOCKER_VERSION)

common-kind-clean: ## Clean up the kind deployment
common-kind-clean:
	-kind delete cluster -n $(KIND_CLUSTER_NAME)


#### Help Target ####

help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'
