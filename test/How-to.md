# How to test the BYO Host provider

This doc provides instructions about how to test BYO Host provider on a local workstation using:

- Kind for provisioning a management cluster
- CAPD provider for creating a workload cluster with control plane nodes only
- Docker run for creating hosts to be used as a capacity for BYO Host machines
- BYO Host provider to add the above hosts to the aforemention workload cluster

## Pre-requisites

It is required to have a docker image to be used when doing docker run for creating hosts

You can fetch a readyt to rool image with Kubernetes v1.19 with:

```shell
docker pull eu.gcr.io/capi-test-270117/byoh/test:v20210510
docker tag eu.gcr.io/capi-test-270117/byoh/test:v20210510 kindest/node:test
```

If instead you want to create your own image, you can use [kinder](https://github.com/kubernetes/kubeadm/tree/master/kinder), a tool used for kubeadm ci testing.

```shell
kinder build node-image-variant --base-image=kindest/base:v20191105-ee880e9b --image=kindest/node:test --with-init-artifacts=v1.19.1 --loglevel=debug
```

## Setting up the management cluster

### Creates the Kubernetes cluster

We are using [kind](https://kind.sigs.k8s.io/) to create the Kubernetes cluster that will be turned into a Cluster API management cluster later in this doc.
Given that we plan to use CAPD, it is required to mount the docker socket into the kind cluster.

```shell
cat > kind-cluster-with-extramounts.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
EOF

kind create cluster --config kind-cluster-with-extramounts.yaml
```

### Installing Cluster API

Installing cluster API into the Kubernetes cluster will turn it into a Cluster API management cluster.

We are going using [tilt](https://tilt.dev/) in order to do so, so you can have your local environment set up for rapid iterations, as described in
[Developing Cluster API with Tilt](https://cluster-api.sigs.k8s.io/developer/tilt.html).

In order to do so you need to clone both https://github.com/kubernetes-sigs/cluster-api/ and https://github.com/vmware-tanzu/cluster-api-provider-byoh locally;
then, from the folder where Cluster-API source code is cloned:

```shell
cat > tilt-settings.json <<EOF
{
  "default_registry": "gcr.io/k8s-staging-cluster-api",
  "enable_providers": ["byoh", "docker", "kubeadm-bootstrap", "kubeadm-control-plane"],
  "provider_repos": ["../../vmware-tanzu/cluster-api-provider-byoh"]
}
EOF

tilt up
```

## Creating the workload cluster

Now that you have a management cluster with Cluster API, CAPD and the BYO Host provider installed, we can start to create a workload
cluster with:

- CAPD based control plane nodes
- BYOH based worker nodes

### Control plane nodes

(from the folder where Cluster-API source code is cloned)

```shell
# from Cluster-API-provider-BYOH folder
export CLUSTER_NAME="test1"
export NAMESPACE="default"
export KUBERNETES_VERSION="v1.19.1"
export CONTROL_PLANE_MACHINE_COUNT=1

cat test/docker-cp.yaml | envsubst | kubectl apply -f -
```

Check if the control plane node is running.

```shell
kubectl get machines 
```

### Add one or more hosts to the capacity pool

(from the folder where Cluster-API source code is cloned)

Create an unmanaged host.

```shell
export HOST_NAME=host1

docker run --detach --tty --hostname $HOST_NAME --name $HOST_NAME --privileged --security-opt seccomp=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /lib/modules:/lib/modules:ro --network kind kindest/node:test
```

Build the agent binary and copy it into the host.

```shell
make release-binaries

docker cp bin/agent-linux-amd64 $HOST_NAME:/agent
```

In order to provide some credentials for the agent to use when to connect to the management cluster, copy management cluster kubeconfig file into the host.

```shell	
cp ~/.kube/config ~/.kube/management-cluster.conf

# on mac OS only, replace host address with the docker internal address
export KIND_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane)
sed -i '' 's/    server\:.*/    server\: https\:\/\/'"$KIND_IP"'\:6443/g' ~/.kube/management-cluster.conf

docker cp ~/.kube/management-cluster.conf $HOST_NAME:/management-cluster.conf
```

Start the agent on the host

```shell
docker exec -it $HOST_NAME bin/bash

./agent --kubeconfig management-cluster.conf
```

Check if the host registered itself into the management cluster.

```shell
kubectl get byohosts 
```

You can now repeat the same steps for additional hosts by changing the HOST_NAME env variable.

### Worker nodes

(from the folder where Cluster-API source code is cloned)

Create a machine deployment with BYOHost

```shell

cat test/byoh-md.yaml | envsubst | kubectl apply -f -
```

Check if the worker node is running.

```shell
kubectl get machines 
```

Dig into ho whe machine gets privisioned.

```shell
kubectl get kubeadmconfig

kubectl get BYOmachines  

kubectl get BYOhost 
```

Or peek at the agent logs.

## Cleanup

```shell
kubectl delete cluster $CLUSTER_NAME
docker rm -f $HOST_NAME
kind delete cluster
```
