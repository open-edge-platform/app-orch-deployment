# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

# Build the app-service-proxy binary
FROM golang:1.24.2-bookworm@sha256:79390b5e5af9ee6e7b1173ee3eac7fadf6751a545297672916b59bfa0ecf6f71 AS build
SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

ARG MAKE_TARGET=go-build
# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

# hadolint ignore=DL3022
COPY --from=root common.mk version.mk /go/src/github.com/open-edge-platform/app-orch-deployment/

WORKDIR /go/src/github.com/open-edge-platform/app-orch-deployment/app-service-proxy

COPY Makefile requirements.txt VERSION go.mod go.sum ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY vendor/ vendor/


RUN CGO_ENABLED=0 \
    TOOL_VERSION_CHECK=0 \
    DOCKER_LABEL_REPO_URL=${REPO_URL} \
    DOCKER_LABEL_VERSION=${VERSION} \
    DOCKER_LABEL_REVISION=${REVISION} \
    DOCKER_LABEL_BUILD_DATE=${BUILD_DATE} \
    GOEXTRAFLAGS=-mod=vendor \
    make ${MAKE_TARGET}


# Use distroless as minimal base image to package the app-service-proxy binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FIXME: Use debug image for development
# From: https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static-debian12:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc AS app-service-proxy

# Run as non-root user
USER nonroot

ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

LABEL org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${REPO_URL} \
      org.opencontainers.image.revision=${REVISION} \
      org.opencontainers.image.created=${BUILD_DATE}

WORKDIR /
COPY --from=build --chown=nonroot:nonroot \
     /go/src/github.com/open-edge-platform/app-orch-deployment/app-service-proxy/bin/app-service-proxy .
COPY web-login /web-login
USER 65532:65532
