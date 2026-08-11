#!/bin/bash

# Copyright 2021 VMware, Inc. All Rights Reserved.
# Copyright 2026 Platform9, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

echo  Update the apt package index and install packages needed to use the Kubernetes apt repository
 apt-get update
 apt-get install -y apt-transport-https ca-certificates curl

echo Download containerd
curl -LOJR https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/cri-containerd-cni-${CONTAINERD_VERSION}-linux-${ARCH}.tar.gz

echo Download the Google Cloud public signing key
 curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://dl.k8s.io/apt/doc/apt-key.gpg

echo Add the Kubernetes apt repository

echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/${KUBERNETES_MAJOR_VERSION}/deb/ /" |  tee /etc/apt/sources.list.d/kubernetes.list

mkdir -p /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/${KUBERNETES_MAJOR_VERSION}/deb/Release.key |  gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo Update apt package index, install kubelet, kubeadm and kubectl
apt-get update
chown -Rv _apt:root /bundle/
chown -R _apt:root /ingredients
mv cri-containerd-cni-${CONTAINERD_VERSION}-linux-${ARCH}.tar.gz /ingredients/
cd /ingredients 
apt-get download {kubelet,kubeadm,kubectl}:$ARCH=$KUBERNETES_VERSION
apt-get download kubernetes-cni:$ARCH
apt-get download cri-tools:$ARCH=$CRITOOL_VERSION
