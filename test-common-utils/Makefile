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


# Versioning variables
VERSION            := $(shell cat VERSION)
GOCMD             ?= go

# Include shared makefile
include ../common.mk

# Security config for Go builds
GOEXTRAFLAGS += $(COMMON_GOEXTRAFLAGS)
#### Development Targets ####

build: go-build ## Build local binaries


# when COPYing --from a named build context
lint: license go-lint #hadolint #mdlint ## Run license, helmlint, go-lint lint tools



#### Sub-targets ####

go-build: ## Build deployment manager binaries
go-build:
	$(GOCMD) build ./...


clean: clean-all common-clean ## Delete all build artifacts


#### Unsupported targets ###

dependency-check-ci: ## Unsupported target
	echo "no dependency check"
