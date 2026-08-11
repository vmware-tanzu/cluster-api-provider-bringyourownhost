# Copyright 2026 Platform9, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Runs repo tests that make real network/Docker calls inside a native Linux
# environment, so they behave the same whether invoked from a Linux CI runner
# or from a macOS workstation's Podman/Docker Desktop VM. See docs/local_dev.md.
FROM golang:1.26.2

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends podman curl \
    && rm -rf /var/lib/apt/lists/* \
    && ln -s "$(command -v podman)" /usr/local/bin/docker

# The e2e suite shells out to kubectl directly (e.g. applying cluster templates,
# dumping cluster state for debugging) rather than going through client-go.
RUN arch="$(dpkg --print-architecture)" \
    && version="$(curl -Ls https://dl.k8s.io/release/stable.txt)" \
    && curl -Lo /usr/local/bin/kubectl "https://dl.k8s.io/release/${version}/bin/linux/${arch}/kubectl" \
    && chmod +x /usr/local/bin/kubectl

WORKDIR /workspace

# Some test code (e.g. CAPI's clusterctl repository generation) shells out to
# "kustomize" via $PATH directly, bypassing the Makefile's own $(KUSTOMIZE)
# absolute-path variable, so the repo's own Makefile-managed tool dirs need
# to be on PATH here too.
ENV PATH="/workspace/bin:/workspace/hack/tools/bin:${PATH}"
