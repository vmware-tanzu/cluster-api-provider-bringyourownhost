#!/bin/bash

# Copyright 2021 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

echo  Update the apt package index and install packages needed to use the Kubernetes apt repository
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl

echo Download containerd
curl -fsSLo containerd.tar https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/cri-containerd-cni-${CONTAINERD_VERSION}-linux-amd64.tar.gz

echo Download the Google Cloud public signing key
curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg

echo Add the Kubernetes apt repository
echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list

echo Update apt package index, install kubelet, kubeadm and kubectl
sudo apt-get update
sudo apt-get download {kubelet,kubeadm,kubectl}:amd64=${KUBERNETES_VERSION}-00
sudo apt-get download kubernetes-cni:$ARCH=1.1.1-00
sudo apt-get download cri-tools:$ARCH=1.25.0-00

echo Strip version to well-known names
# Mandatory
cp *kubeadm*.deb ./kubeadm.deb
cp *kubelet*.deb ./kubelet.deb
cp *kubectl*.deb ./kubectl.deb
# Optional
cp *cri-tools*.deb cri-tools.deb > /dev/null | true
cp *kubernetes-cni*.deb kubernetes-cni.deb > /dev/null | true