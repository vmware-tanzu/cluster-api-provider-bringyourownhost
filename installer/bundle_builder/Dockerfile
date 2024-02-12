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

# FROM harbor-repo.vmware.com/dockerhub-proxy-cache/k14s/image
FROM ubuntu:20.04
# If set to 1 bundle is built and available as bundle/bundle.tar
# If set to 0 bundle is build and pushed to repo
ENV BUILD_ONLY=1

WORKDIR /bundle-builder
COPY *.sh ./
RUN chmod a+x *.sh
#Default config
COPY config/ubuntu/20_04/k8s/1_22 /config/

# Install imgpkg
# RUN echo "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable" >> /etc/apt/source.list
COPY kubernetes.list /etc/apt/sources.list.d/
RUN apt-get -o "Acquire::https::Verify-Peer=false" -o "APT::Get::AllowUnauthenticated=true" update && DEBIAN_FRONTEND=noninteractive apt-get -o "Acquire::https::Verify-Peer=false"  -o "APT::Get::AllowUnauthenticated=true" install -y --no-install-recommends perl wget docker-ce-cli ca-certificates yc-infra-crt-nemax yc-infra-crt-nebius
# RUN apt-get -o "Acquire::https::Verify-Peer=false" update --allow-unauthenticated --allow-insecure-repositories && DEBIAN_FRONTEND=noninteractive apt-get install -y --allow-unauthenticated --no-install-recommends --allow-insecure-repositories perl wget docker-ce-cli ca-certificates yc-infra-crt-nemax yc-infra-crt-nebius
RUN mkdir /local-bin && K14SIO_INSTALL_BIN_DIR=/local-bin /usr/bin/bash /bundle-builder/install_imgpkg.sh

RUN mkdir /ingredients && mkdir /bundle
ENV PATH="/local-bin:/bundle-builder:${PATH}"

WORKDIR /tmp/bundle
ENTRYPOINT ["build-push-bundle.sh", "/ingredients", "/config"]
