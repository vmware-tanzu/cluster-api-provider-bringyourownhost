# Copyright 2021 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Build and push (opt-in) a BYOH bundle to repository
#
# 1. Download bundle ingredients. See ingredients/deb/download.sh
# 2. Mount the bundle ingredients under /ingredients
# 3. Optional. Mount output bundle directory under /bundle
# 3. Optional. Mount additional configuration under /config
#	-v config/ubuntu/20_04/k8s/1_22:/config
#	Defaults to config/ubuntu/20_04/k8s/1_22
# Example
# // Build and push a BYOH bundle to repository
# docker run --rm -v <INGREDIENTS_HOST_ABS_PATH>:/ingredients --env BUILD_ONLY=0 <THIS_IMAGE> <REPO>/<BUNDLE IMAGE>
#
# // Build and store a BYOH bundle
# docker run --rm -v <INGREDIENTS_HOST_ABS_PATH>:/ingredients  -v <BUNDLE_OUTPUT_ABS_PATH>:/bundle --env <THIS_IMAGE>

ARG BASE_OS=ubuntu
ARG BASE_OS_VERSION=20.04

FROM $BASE_OS:$BASE_OS_VERSION as build


ARG KUBERNETES_VERSION=1.21.8
ENV BASE_OS=$BASE_OS
# Override to download other version
ENV CONTAINERD_VERSION=1.6.8
ENV ARCH=amd64

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends sudo


WORKDIR /tmp
COPY ./config/ubuntu/20_04/k8s/1_22/ /tmp/config/
RUN cd /tmp/config/ && tar -czvf conf.tar *
RUN mkdir /bundle-builder && mv /tmp/config/conf.tar /bundle-builder/conf.tar

WORKDIR /bundle-builder
COPY download.sh .
RUN chmod a+x download.sh
RUN ./download.sh
RUN rm download.sh
RUN tar -cvf bundle.tar *

FROM scratch
COPY --from=build /bundle-builder/kubernetes-cni.deb /kubernetes-cni.deb
COPY --from=build /bundle-builder/kubelet.deb /kubelet.deb
COPY --from=build /bundle-builder/kubectl.deb /kubectl.deb
COPY --from=build /bundle-builder/kubeadm.deb /kubeadm.deb
COPY --from=build /bundle-builder/cri-tools.deb /cri-tools.deb
COPY --from=build /bundle-builder/containerd.tar /containerd.tar
COPY --from=build /bundle-builder/containerd.tar /containerd.tar
COPY --from=build /bundle-builder/conf.tar /conf.tar
COPY --from=build /bundle-builder/bundle.tar /bundle.tar