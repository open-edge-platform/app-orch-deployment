#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_ROOT="${SCRIPT_DIR}/.."
CODEGEN_PKG="${CODEGEN_PKG:-"${SCRIPT_ROOT}/../../code-generator"}"

source "${CODEGEN_PKG}/kube_codegen.sh"

THIS_PKG="github.com/open-edge-platform/app-orch-deployment/app-interconnect"

kube::codegen::gen_helpers \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}"

if [[ -n "${API_KNOWN_VIOLATIONS_DIR:-}" ]]; then
    report_filename="${API_KNOWN_VIOLATIONS_DIR}/codegen_violation_exceptions.list"
    if [[ "${UPDATE_API_KNOWN_VIOLATIONS:-}" == "true" ]]; then
        update_report="--update-report"
    fi
fi

kube::codegen::gen_client \
    --with-watch \
    --output-dir "${SCRIPT_ROOT}/pkg" \
    --output-pkg "${THIS_PKG}/pkg" \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}/pkg/apis"
