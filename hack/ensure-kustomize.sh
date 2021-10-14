#!/usr/bin/env bash

# Copyright 2021 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
BIN_ROOT="${KUBE_ROOT}/hack/tools/bin"

kustomize_version=3.9.1

goarch=amd64
goos="unknown"
if [[ "${OSTYPE}" == "linux"* ]]; then
  goos="linux"
elif [[ "${OSTYPE}" == "darwin"* ]]; then
  goos="darwin"
fi

if [[ "$goos" == "unknown" ]]; then
  echo "OS '$OSTYPE' not supported. Aborting." >&2
  exit 1
fi

# Ensure the kustomize tool exists and is a viable version, or installs it
verify_kustomize_version() {
  if ! [ -x "$(command -v "${BIN_ROOT}/kustomize")" ]; then
    echo "fetching kustomize@${kustomize_version}"
    if ! [ -d "${BIN_ROOT}" ]; then
      mkdir -p "${BIN_ROOT}"
    fi
    archive_name="kustomize-v${kustomize_version}.tar.gz"
    curl -sLo "${BIN_ROOT}/${archive_name}" https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${kustomize_version}/kustomize_v${kustomize_version}_${goos}_${goarch}.tar.gz
    tar -zvxf "${BIN_ROOT}/${archive_name}" -C "${BIN_ROOT}/"
    chmod +x "${BIN_ROOT}/kustomize"
    rm "${BIN_ROOT}/${archive_name}"
  fi
}

verify_kustomize_version
