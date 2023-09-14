# Copyright 2021 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Downloads bundle ingredients : containerd as tar, kubelet, kubeadm, kubectl as Debian packages
#
# Usage:
# 1. Mount a host path as /ingredients
# 2. Run the image
#

ARG BASE_IMAGE=rockylinux:8.7
FROM $BASE_IMAGE as build

# Override to download other version
ENV CONTAINERD_VERSION=1.6.18
ENV KUBERNETES_VERSION=1.25.11-0
ENV ARCH=x86_64

RUN dnf -y install sudo

WORKDIR /bundle-builder
COPY download.sh .
RUN chmod a+x download.sh
WORKDIR /ingredients

ENTRYPOINT ["/bundle-builder/download.sh"]
