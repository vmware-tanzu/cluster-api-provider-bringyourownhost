#!/bin/bash

# Copyright 2021 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

echo  Update the apt package index and install packages needed to use the Kubernetes apt repository
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl

echo Download containerd
curl -LOJR https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/cri-containerd-cni-${CONTAINERD_VERSION}-linux-amd64.tar.gz

echo Download the Google Cloud public signing key
curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg

echo Add the Kubernetes apt repository
echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list

echo Update apt package index, install kubelet, kubeadm and kubectl
sudo apt-get update
sudo apt-get download {kubelet,kubeadm,kubectl}:$ARCH=$KUBERNETES_VERSION
sudo apt-get download kubernetes-cni:$ARCH=0.8.7-00
sudo apt-get download cri-tools:$ARCH=1.23.0-00
