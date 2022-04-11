# Getting Started

This is a guide on how to get started with Cluster API Provider BringYourOwnHost(BYOH). To learn more about Cluster API in more depth, check out the [Cluster API book][cluster-api-book].

## Install Requirements
- clusterctl, which can be downloaded from the latest [release][releases] of Cluster API (CAPI) on GitHub.
- [Kind][kind] can be used  to provide an initial management cluster for testing.
- [kubectl][kubectl] is required to access your workload clusters.
- Ubuntu 20.04 and above (Linux Kernel 5.4 and above) is required for accessing kernel configs during kubeadm preflight checks.

## Create a management cluster
Cluster API requires an existing Kubernetes cluster accessible via kubectl. During the installation process the
Kubernetes cluster will be transformed into a [management cluster](https://cluster-api.sigs.k8s.io/reference/glossary.html#management-cluster) by installing the Cluster API [provider components](https://cluster-api.sigs.k8s.io/reference/glossary.html#provider-components), so it is recommended to keep it separated from any application workload.

**Choose one of the options below:**

1. **Existing Management Cluster**

    For production use-cases a "real" Kubernetes cluster should be used with appropriate backup and DR policies and procedures in place. The Kubernetes cluster must be at least v1.19.1.

    ```bash
    export KUBECONFIG=<...>
    ```
**OR**

2. **Kind**

    If you are testing locally, you can use [Kind][kind] to create a cluster with the following command:

    ```shell
    kind create cluster
    ```

---
[kind] is not designed for production use.

**Minimum [kind] supported version**: v0.9.0

Note for macOS users: you may need to [increase the memory available](https://docs.docker.com/docker-for-mac/#resources) for containers (recommend 6Gb for CAPD).

---

## Initialize the management cluster and install the BringYourOwnHost provider

Now that we've got clusterctl installed and all the prerequisites in place, let's transform the Kubernetes cluster
into a management cluster by using `clusterctl init`.

```shell
clusterctl init --infrastructure byoh
```

## Creating a BYOH workload cluster
 
Once the management cluster is ready, you will need to create a few hosts that the `BringYourOwnHost` provider can use, before you can create your first workload cluster.

If you already have hosts (these could be bare metal servers / VMs / containers) ready, then please skip to [Register BYOH host to management cluster](#register-byoh-host-to-management-cluster)

If not, you could create containers to deploy your workload clusters on. We have a `make` task that will create a docker image for you locally to start. 

```shell
cd cluster-api-provider-bringyourownhost
make prepare-byoh-docker-host-image
```

Once the image is ready, lets start 2 docker containers for our deployment. One for the control plane, and one for the worker. (you could of course run more)

```shell
for i in {1..2}
do
  echo "Creating docker container named host$i"
  docker run --detach --tty --hostname host$i --name host$i --privileged --security-opt seccomp=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /lib/modules:/lib/modules:ro --network kind byoh/node:e2e
done
```

## Register BYOH host to management cluster

---
### VM Prerequisites
- The following packages must be pre-installed on the VMs
  - socat
  - ebtables
  - ethtool
  - conntrack
- You can install these with
``` shell
sudo apt-get install socat ebtables ethtool conntrack
```
- The output of `hostname` should be added to `/etc/hosts`

Example:
```shell
$ hostname
node01

$ cat /etc/hosts
127.0.0.1 localhost
127.0.0.1 node01
...
```

If you are trying this on your own hosts, then for each host
1. Download the [byoh-hostagent-linux-amd64](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/releases/download/v0.2.0/byoh-hostagent-linux-amd64)
2. Copy the management cluster `kubeconfig` file as `management.conf`
3. Start the agent 
```shell
./byoh-hostagent-linux-amd64 -kubeconfig management-cluster.conf > byoh-agent.log 2>&1 &
```

---
If you are trying this using the docker containers we started above, then we would first need to prep the kubeconfig to be used from the docker containers. By default, the kubeconfig states that the server is at `127.0.0.1`. We need to swap this out with the kind container IP. 

```shell
cp ~/.kube/config ~/.kube/management-cluster.conf
export KIND_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane)
sed -i 's/    server\:.*/    server\: https\:\/\/'"$KIND_IP"'\:6443/g' ~/.kube/management-cluster.conf
```
Assuming you have downloaded the `byoh-hostagent-linux-amd64` into your working directory, you can use the following script to start the agent on the containers.

```shell
for i in {1..2}
do
echo "Copy agent binary to host $i"
docker cp byoh-hostagent-linux-amd64 host$i:/byoh-hostagent
echo "Copy kubeconfig to host $i"
docker cp ~/.kube/management-cluster.conf host$i:/management-cluster.conf
done
```
Start the host agent on each of the hosts and keep it running. Use the `--skip-installation` flag if you already have k8s components included in the docker image. This flag will skip k8s installation attempt on the host

```shell
export HOST_NAME=host1
docker exec -it $HOST_NAME sh -c "chmod +x byoh-hostagent && ./byoh-hostagent --kubeconfig management-cluster.conf"

# do the same for host2 in a separate tab
export HOST_NAME=host2
docker exec -it $HOST_NAME sh -c "chmod +x byoh-hostagent && ./byoh-hostagent --kubeconfig management-cluster.conf"
```
---

You should be able to view your registered hosts using

```shell
kubectl get byohosts
```

## Create workload cluster
Running the following command(on the host where you execute `clusterctl` in previous steps)

**NOTE:** The `CONTROL_PLANE_ENDPOINT_IP` is an IP that must be an IP on the same subnet as the control plane machines, it should be also an IP that is not part of your DHCP range.

If you are using docker containers then you can find the control plane machine's network subnet by running

```shell
docker network inspect kind | jq -r 'map(.IPAM.Config[].Subnet) []'
```
Randomly assign any free IP within the network subnet to the `CONTROL_PLANE_ENDPOINT_IP`. The list of IP addresses currently in use can be found by

```shell
docker network inspect kind | jq -r 'map(.Containers[].IPv4Address) []'
```

### Create the workload cluster
Generate the cluster.yaml for workload cluster
 - for vms as byohosts
    ```shell
    CONTROL_PLANE_ENDPOINT_IP=10.10.10.10 clusterctl generate cluster byoh-cluster \
      --infrastructure byoh \
      --kubernetes-version v1.22.3 \
      --control-plane-machine-count 1 \
      --worker-machine-count 1 > cluster.yaml
    ```

 - for docker hosts use the --flavor argument
    ```shell
    CONTROL_PLANE_ENDPOINT_IP=10.10.10.10 clusterctl generate cluster byoh-cluster \
        --infrastructure byoh \
        --kubernetes-version v1.22.3 \
        --control-plane-machine-count 1 \
        --worker-machine-count 1 \
        --flavor docker > cluster.yaml
    ```

Inspect and make any changes
```shell
# for vms as byohosts
$ BUNDLE_LOOKUP_TAG=v1.23.5 CONTROL_PLANE_ENDPOINT_IP=10.10.10.10 clusterctl generate cluster byoh-cluster \
    --infrastructure byoh \
    --kubernetes-version v1.23.5 \
    --control-plane-machine-count 1 \
    --worker-machine-count 1 > cluster.yaml

# for docker hosts use the --flavor argument
$ BUNDLE_LOOKUP_TAG=v1.23.5 CONTROL_PLANE_ENDPOINT_IP=10.10.10.10 clusterctl generate cluster byoh-cluster \
    --infrastructure byoh \
    --kubernetes-version v1.23.5 \
    --control-plane-machine-count 1 \
    --worker-machine-count 1 \
    --flavor docker > cluster.yaml

# Inspect and make any changes
$ vi cluster.yaml

Create the workload cluster in the current namespace on the management cluster
```shell
kubectl apply -f cluster.yaml
```

## Accessing the workload cluster

The `kubeconfig` for the workload cluster will be stored in a secret, which can
be retrieved using:

``` shell
kubectl get secret/byoh-cluster-kubeconfig -o json \
  | jq -r .data.value \
  | base64 --decode \
  > ./byoh-cluster.kubeconfig
```

The kubeconfig can then be used to apply a CNI for networking, for example, Calico:

```shell
KUBECONFIG=byoh-cluster.kubeconfig kubectl apply -f https://docs.projectcalico.org/v3.20/manifests/calico.yaml
```

after that you should see your nodes turn into ready:

```shell
$ KUBECONFIG=byoh-cluster.kubeconfig kubectl get nodes
NAME                                                          STATUS     ROLES    AGE   VERSION
byoh-cluster-8siai8                                           Ready      master   5m   v1.23.5

```



<!-- References -->
[cluster-api-book]: https://cluster-api.sigs.k8s.io/
[glossary-bootstrapping]: https://cluster-api.sigs.k8s.io/reference/glossary.html#bootstrap
[kind]: https://kind.sigs.k8s.io
[glossary-management-cluster]: https://github.com/kubernetes-sigs/cluster-api/blob/master/docs/book/GLOSSARY.md#management-cluster
[releases]: https://github.com/kubernetes-sigs/cluster-api/releases
[docker]: https://docs.docker.com/glossary/?term=install
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
