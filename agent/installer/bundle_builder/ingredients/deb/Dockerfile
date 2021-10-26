# Downloads containerd as tar, kubelet, kubeadm, kubectl as Debian packages
#
# Usage:
# 1. Mount a host path as /payload
# 2. Run the image
#

ARG BASE_IMAGE=ubuntu:18.04
FROM $BASE_IMAGE as build

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends sudo

WORKDIR /bundle-builder
COPY download.sh .
RUN chmod a+x download.sh
WORKDIR /payload

ENTRYPOINT ["/bundle-builder/download.sh"]
