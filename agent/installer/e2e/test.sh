#~/bin/bash

# host machine is accessible as gateway by guest
REPO_IP=$(/sbin/ip route | awk '/default/ { print $3 }')
REPO_PORT=$1
K8SVER=$2

echo "===Install Pre-requisites"
apt-get install socat ebtables ethtool conntrack -y

set -e

echo "===Run Install"
./cli --install --bundle-repo $REPO_IP:$REPO_PORT --k8s $K8SVER

echo "===Run kubeadm preflight"
kubeadm init phase preflight --ignore-preflight-errors=Cpu,Mem

echo "===Run Uninstall"
./cli --uninstall --bundle-repo $REPO_IP:$REPO_PORT --k8s $K8SVER

