# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: help build test lint

# Optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 1

##### Variables #####

# Required Go Dependencies
GOLINT        := true
GOCMD             ?= go

# Versioning variables
VERSION            := $(shell cat VERSION)


# Include shared makefile
include ../common.mk

# Security config for Go builds
GOEXTRAFLAGS += $(COMMON_GOEXTRAFLAGS)
#### Development Targets ####

build: go-build ## Build local binaries
go-build:
go-build:
	$(GOCMD) build ./...

lint: license go-lint # Run license, helmlint, go-lint lint tools

clean: clean-all common-clean ## Delete all build artifacts


#### Unsupported targets ###

dependency-check-ci: ## Unsupported target
	echo "no dependency check"
