#!/bin/bash

CONTAINERD_VERSION=1.5.7
KUBUERNETES_VERSION=1.21.2-00
ARCH=amd64

set -e

echo  Update the apt package index and install packages needed to use the Kubernetes apt repository
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl

echo Download containerd
curl -OJ https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/cri-containerd-cni-${CONTAINERD_VERSION}-linux-amd64.tar.gz

echo Download the Google Cloud public signing key
curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg

echo Add the Kubernetes apt repository
echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list

echo Update apt package index, install kubelet, kubeadm and kubectl
sudo apt-get update
sudo apt-get download {kubelet,kubeadm,kubectl}:$ARCH=$KUBUERNETES_VERSION
