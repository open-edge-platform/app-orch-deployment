# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.9@sha256:5034fa44b36163a4109b71ed75c67dbdcb52c3cd9750953befe00054315d9fd2 AS build

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

ARG MAKE_TARGET=go-build

# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

# hadolint ignore=DL3022
COPY --from=root common.mk version.mk /go/src/github.com/open-edge-platform/app-orch-deployment/

WORKDIR /go/src/github.com/open-edge-platform/app-orch-deployment/app-resource-manager

COPY Makefile requirements.txt VERSION go.mod go.sum ./
COPY api/nbi/ ./api/nbi/
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY test/ ./test/
COPY vendor/ ./vendor/

RUN CGO_ENABLED=0 \
    TOOL_VERSION_CHECK=0 \
    DOCKER_LABEL_REPO_URL=${REPO_URL} \
    DOCKER_LABEL_VERSION=${VERSION} \
    DOCKER_LABEL_REVISION=${REVISION} \
    DOCKER_LABEL_BUILD_DATE=${BUILD_DATE} \
    GOEXTRAFLAGS=-mod=vendor \
    make ${MAKE_TARGET}

# From: https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static-debian12:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc AS app-resource-manager

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

COPY --from=build --chown=nonroot:nonroot \
     /go/src/github.com/open-edge-platform/app-orch-deployment/app-resource-manager/bin/app-resource-manager \
     /usr/local/bin/

ENTRYPOINT ["app-resource-manager"]

# From: https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static-debian12:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc AS vnc-proxy

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

COPY --from=build --chown=nonroot:nonroot \
     /go/src/github.com/open-edge-platform/app-orch-deployment/app-resource-manager/bin/vnc-proxy \
     /usr/local/bin/

COPY vnc-proxy-web-ui /usr/local/html/vnc-proxy-web-ui/

ENTRYPOINT ["vnc-proxy"]

# From: https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static-debian12:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc AS rest-proxy

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

COPY --from=build --chown=nonroot:nonroot \
     /go/src/github.com/open-edge-platform/app-orch-deployment/app-resource-manager/bin/rest-proxy \
     /usr/local/bin/
COPY --from=build --chown=nonroot:nonroot \
     /go/src/github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/spec/v2/openapi.yaml \
     /usr/local/etc/v2/openapi.yaml

COPY --from=build --chown=nonroot:nonroot \
      /go/src/github.com/open-edge-platform/app-orch-deployment/app-resource-manager/bin/app-resource-manager \
      /usr/local/bin/

COPY --from=build --chown=nonroot:nonroot \
     /go/src/github.com/open-edge-platform/app-orch-deployment/app-resource-manager/bin/vnc-proxy \
     /usr/local/bin/

ENTRYPOINT ["rest-proxy"]
