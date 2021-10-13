# How to test the BYO Host provider

This doc provides instructions about how to test BYO Host provider on a local workstation using:

- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) for provisioning a management cluster
- [CAPD](https://github.com/kubernetes-sigs/cluster-api/tree/master/test/infrastructure/docker) provider for creating a workload cluster with control plane nodes only
- [Docker](https://docs.docker.com/engine/install/) run for creating hosts to be used as a capacity for BYO Host machines
- [BYOH](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost) provider to add the above hosts to the aforemention workload cluster
- [Tilt](https://docs.tilt.dev/install.html) for faster iterative development

## Pre-requisites

It is required to have a docker image to be used when doing docker run for creating hosts

__Clone BYOH Repo__
```shell
git clone git@github.com:vmware-tanzu/cluster-api-provider-bringyourownhost.git
```

__Build image__
```shell
cd cluster-api-provider-bringyourownhost
make prepare-byoh-image
```

## Setting up the management cluster

### Creates the Kubernetes cluster

We are using [kind](https://kind.sigs.k8s.io/) to create the Kubernetes cluster that will be turned into a Cluster API management cluster later in this doc.

```shell
cat > kind-cluster.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1beta1
nodes:
- role: control-plane
  image: kindest/node:v1.22.0
EOF

kind create cluster --config kind-cluster.yaml
```

### Installing Cluster API

Installing cluster API into the Kubernetes cluster will turn it into a Cluster API management cluster.

We are going using [tilt](https://tilt.dev/) in order to do so, so you can have your local environment set up for rapid iterations, as described in
[Developing Cluster API with Tilt](https://cluster-api.sigs.k8s.io/developer/tilt.html).

In order to do so you need to clone both https://github.com/kubernetes-sigs/cluster-api/ and https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost locally;

__Clone CAPI Repo__

```shell
git clone git@github.com:kubernetes-sigs/cluster-api.git
cd cluster-api
git checkout v1.0.0 
```

__Create a tilt-settings.json file__

Next, create a tilt-settings.json file and place it in your local copy of cluster-api:  

```shell
cat > tilt-settings.json <<EOF
{
  "default_registry": "gcr.io/k8s-staging-cluster-api",
  "enable_providers": ["byoh", "kubeadm-bootstrap", "kubeadm-control-plane"],
  "provider_repos": ["../cluster-api-provider-bringyourownhost"]
}
EOF
```

__Run Tilt__

To launch your development environment, run below command and keep it running in the shell

```shell
tilt up
```
Wait for all the resources to come up, status can be viewed in Tilt UI.
## Creating the workload cluster

Now that you have a management cluster with Cluster API and BYOHost provider installed, we can start to create a workload
cluster.

### Add a minimum of two hosts to the capacity pool

Create an unmanaged host.

```shell
export HOST_NAME=host1

docker run --detach --tty --hostname $HOST_NAME --name $HOST_NAME --privileged --security-opt seccomp=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /lib/modules:/lib/modules:ro --network kind byoh/node:v1.22.0
```

Build the agent binary and copy it into the host.

```shell
# from cluster-api-provider-bringyourownhost folder
make host-agent-binaries

docker cp bin/agent-linux-amd64 $HOST_NAME:/agent
```

In order to provide some credentials for the agent to use when to connect to the management cluster, copy management cluster kubeconfig file into the host.

```shell	
cp ~/.kube/config ~/.kube/management-cluster.conf

export KIND_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane)
sed -i 's/    server\:.*/    server\: https\:\/\/'"$KIND_IP"'\:6443/g' ~/.kube/management-cluster.conf

docker cp ~/.kube/management-cluster.conf $HOST_NAME:/management-cluster.conf
```

Start the agent on the host and keep it running

```shell
docker exec -it $HOST_NAME bin/bash

./agent --kubeconfig management-cluster.conf
```

Repeat the same steps with by changing the `HOST_NAME` env variable.

Check if the hosts registered itself into the management cluster.

Open another shell and run
```shell
kubectl get byohosts 
```

Open a new shell and change directory to `cluster-api-provider-byoh` repository. Run below commands

```shell
export CLUSTER_NAME="test1"
export NAMESPACE="default"
export KUBERNETES_VERSION="v1.22.0"
export CONTROL_PLANE_MACHINE_COUNT=1
export WORKER_MACHINE_COUNT=1
export CONTROL_PLANE_ENDPOINT=<static IP from the subnet where the containers are running>
export VIP_INTERFACE=<interface on the container>
```

# from cluster-api-provider-bringyourownhost folder

```shell
cat test/e2e/data/infrastructure-provider-byoh/v1beta1/cluster-template-byoh.yaml | envsubst | kubectl apply -f -
```

```shell
kubectl get machines 
```

Dig into host when machine gets provisioned.

```shell
kubectl get kubeadmconfig

kubectl get BYOmachines  

kubectl get BYOhost 
```

Deploy a CNI solution

```shell
kubectl get secret $CLUSTER_NAME-kubeconfig -o jsonpath='{.data.value}' | base64 -d > $CLUSTER_NAME-kubeconfig
kubectl --kubeconfig $CLUSTER_NAME-kubeconfig apply -f test/e2e/data/cni/kindnet/kindnet.yaml
```

After a short while, our nodes should be running and in Ready state.
Check the workload cluster

```shell
kubectl --kubeconfig $CLUSTER_NAME-kubeconfig get nodes
```

Or peek at the agent logs.
## Cleanup

```shell
kubectl delete cluster $CLUSTER_NAME
docker rm -f $HOST_NAME
kind delete cluster
```
