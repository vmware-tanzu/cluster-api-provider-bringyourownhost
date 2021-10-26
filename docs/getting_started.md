# Getting Started

This is a guide on how to get started with Cluster API Provider BringYourOwnHost. To learn more about cluster API in more
depth, check out the the [Cluster API book][cluster-api-book].



## Install Requirements

- clusterctl, which can be downloaded from the latest [release][releases] of Cluster API (CAPI) on GitHub.
- [Kind][kind] can be used  to provide an initial management cluster for testing.
- [kubectl][kubectl] is required to access your workload clusters.


## Create a management cluster
BringYourOwnHost Cluster API Provider requires an existing Kubernetes cluster accessible via `kubectl`.
### Existing cluster
If you already have a Kubernetes cluster, appropriate backup procedures should be in place before your take any actions.
```shell
export KUBECONFIG=<...>

````
### Kind cluster
If you are testing locally, you can use [Kind][kind] create a cluster with the following command:

[Docker][docker] is required for using `kind`.
```shell
kind create cluster
```

## Configuring and installing BringYourOwnHost provider in a management cluster

To initialize Cluster API Provider BringYourOwnHost, clusterctl requires the following settings, which should
be set in `~/.cluster-api/clusterctl.yaml` as the following:

``` yaml
providers:
  - name: byoh
    url: https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/releases/latest/infrastructure-components.yaml
    type: InfrastructureProvider                                                              
```


running `clusterctl config repositories`.

You should be able to see the new BYOH provider there.
```shell
clusterctl config repositories
NAME           TYPE                     URL                                                                                          FILE
cluster-api    CoreProvider             https://github.com/kubernetes-sigs/cluster-api/releases/latest/                              core-components.yaml
...
byoh           InfrastructureProvider   https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/releases/latest/       infrastructure-components.yaml
...
vsphere        InfrastructureProvider   https://github.com/kubernetes-sigs/cluster-api-provider-vsphere/releases/latest/             infrastructure-components.yaml
```

Install the BYOH provider

```shell
clusterctl init --infrastructure byoh
```

## Creating a BYOH workload cluster

### Register BYOH host to management cluster.

On each BYOH host

1. Download the [byoh-hostagent-linux-amd64](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/releases/latest) 
2. Save the management cluster kubeconfig file as management.conf
3. Start the agent 
```shell
./byoh-hostagent-linux-amd64 -kubeconfig management.conf > byoh-agent.log 2>&1 &
```

### Create workload cluster
Running the following command(on the host where you execute clusterctl in previous steps)

**NOTE:** The CONTROL_PLANE_ENDPOINT_IP is an IP that must be an IP on the same subnet as the control plane machines, it should be also an IP that is not part of your DHCP range

```shell
$ CONTROL_PLANE_ENDPOINT_IP=10.10.10.10 clusterctl generate cluster byoh-cluster \
    --infrastructure byoh \
    --kubernetes-version v1.21.2+vmware.1 \
    --control-plane-machine-count 1 \
    --worker-machine-count 3 > cluster.yaml

# Inspect and make any changes
$ vi cluster.yaml

# Create the workload cluster in the current namespace on the management cluster
$ kubectl apply -f cluster.yaml
```


## Accessing the workload cluster

The kubeconfig for the workload cluster will be stored in a secret, which can
be retrieved using:

``` shell
$ kubectl get secret/byoh-cluster-kubeconfig -o json \
  | jq -r .data.value \
  | base64 --decode \
  > ./byoh-cluster.kubeconfig
```

The kubeconfig can then be used to apply a CNI for networking, for example, Calico:

```shell
KUBECONFIG=byoh-cluster.kubeconfig kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
```

after that you should see your nodes turn into ready:

```shell
$ KUBECONFIG=byoh-cluster.kubeconfig kubectl get nodes
NAME                                                          STATUS     ROLES    AGE   VERSION
byoh-cluster-8siai8                                      Ready      master   5m   v1.21.2

```



<!-- References -->
[cluster-api-book]: https://cluster-api.sigs.k8s.io/
[glossary-bootstrapping]: https://cluster-api.sigs.k8s.io/reference/glossary.html#bootstrap
[kind]: https://kind.sigs.k8s.io
[glossary-management-cluster]: https://github.com/kubernetes-sigs/cluster-api/blob/master/docs/book/GLOSSARY.md#management-cluster
[releases]: https://github.com/kubernetes-sigs/cluster-api/releases
[docker]: https://docs.docker.com/glossary/?term=install
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
