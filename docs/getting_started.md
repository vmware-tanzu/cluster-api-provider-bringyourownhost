# Getting Started

This is a guide on how to get started with Cluster API Provider BringYourOwnHost. To learn more about cluster API in more
depth, check out the the [Cluster API book][cluster-api-book].

- [Getting Started](#getting-started)
    - [Install Requirements](#install-requirements)
        - [vSphere Requirements](#vsphere-requirements)
            - [vCenter Credentials](#vcenter-credentials)
            - [Uploading the machine images](#uploading-the-machine-images)
    - [Creating a test management cluster](#creating-a-test-management-cluster)
    - [Configuring and installing Cluster API Provider vSphere in a management cluster](#configuring-and-installing-cluster-api-provider-vsphere-in-a-management-cluster)
    - [Creating a vSphere-based workload cluster](#creating-a-vsphere-based-workload-cluster)
    - [Accessing the workload cluster](#accessing-the-workload-cluster)

## Install Requirements

- clusterctl, which can downloaded the latest [release][releases] of Cluster API (CAPI) on GitHub.
- [Docker][docker] is required for the bootstrap cluster using `clusterctl`.
- [Kind][kind] can be used  to provide an initial management cluster for testing.
- [kubectl][kubectl] is required to access your workload clusters.


## Configure a Kubernetes cluster
BringYourOwnHost cluster api provider require an existing Kubernetes cluster accessible via kubectl.
### Existing cluster
If you already have a Kubernetes cluster, appropriate backup should be procedures in place before your take any actions.
```shell
export KUBECONFIG=<...>

````
### Kind cluster
If you are testing locally, you can use [Kind][kind] create a cluster with the following command:

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
kind create cluster --config kind-cluster-with-extraMounts.yaml
```

## Configuring and installing Cluster API Provider vSphere in a management cluster

To initialize Cluster API Provider vSphere, clusterctl requires the following variables, which should
be set in `~/.cluster-api/clusterctl.yaml` as the following:

``` yaml
## -- Controller settings -- ##
VSPHERE_USERNAME: "vi-admin@vsphere.local"                    # The username used to access the remote vSphere endpoint
VSPHERE_PASSWORD: "admin!23"                                  # The password used to access the remote vSphere endpoint

## -- Required workload cluster default settings -- ##
VSPHERE_SERVER: "10.0.0.1"                                    # The vCenter server IP or FQDN
VSPHERE_DATACENTER: "SDDC-Datacenter"                         # The vSphere datacenter to deploy the management cluster on
VSPHERE_DATASTORE: "DefaultDatastore"                         # The vSphere datastore to deploy the management cluster on
VSPHERE_NETWORK: "VM Network"                                 # The VM network to deploy the management cluster on
VSPHERE_RESOURCE_POOL: "*/Resources"                          # The vSphere resource pool for your VMs
VSPHERE_FOLDER: "vm"                                          # The VM folder for your VMs. Set to "" to use the root vSphere folder
VSPHERE_TEMPLATE: "ubuntu-1804-kube-v1.17.3"                  # The VM template to use for your management cluster.
CONTROL_PLANE_ENDPOINT_IP: "192.168.9.230"                    # the IP that kube-vip is going to use as a control plane endpoint
VSPHERE_TLS_THUMBPRINT: "..."                                 # sha1 thumbprint of the vcenter certificate: openssl x509 -sha1 -fingerprint -in ca.crt -noout
EXP_CLUSTER_RESOURCE_SET: "true"                              # This enables the ClusterResourceSet feature that we are using to deploy CSI
VSPHERE_SSH_AUTHORIZED_KEY: "ssh-rsa AAAAB3N..."              # The public ssh authorized key on all machines
                                                              #   in this cluster.
                                                              #   Set to "" if you don't want to enable SSH,
                                                              #   or are using another solution.
VSPHERE_STORAGE_POLICY: ""                                    # This is the vSphere storage policy.
                                                              #  Set it to "" if you don't want to use a storage policy.
```

If you are using the **DEPRECATED** `haproxy` flavour you will need to add the following variable to your `clusterctl.yaml`:

```yaml
VSPHERE_HAPROXY_TEMPLATE: "capv-haproxy-v0.6.4"               # The VM template to use for the HAProxy load balancer
```

**NOTE**: Technically, SSH keys and vSphere folders are optional, but optional template variables are not currently
supported by clusterctl. If you need to not set the vSphere folder or SSH keys, then remove the appropriate fields after
running `clusterctl config`.

the `CONTROL_PLANE_ENDPOINT_IP` is an IP that must be an IP on the same subnet as the control plane machines, it should be also an IP that is not part of your DHCP range

`CONTROL_PLANE_ENDPOINT_IP` is mandatory when you are using the default and the `external-loadbalancer` flavour

the `EXP_CLUSTER_RESOURCE_SET` is required if you want to deploy CSI using cluster resource sets (mandatory in the default flavor).

Setting `VSPHERE_USERNAME` and `VSPHERE_PASSWORD` is one way to manage identities. For the full set of options see [identity management](identity_management.md).

Once you have access to a management cluster, you can instantiate Cluster API with the following:

```shell
clusterctl init --infrastructure vsphere
```

## Creating a vSphere-based workload cluster

The following command

```shell
$ clusterctl config cluster vsphere-quickstart \
    --infrastructure vsphere \
    --kubernetes-version v1.17.3 \
    --control-plane-machine-count 1 \
    --worker-machine-count 3 > cluster.yaml

# Inspect and make any changes
$ vi cluster.yaml

# Create the workload cluster in the current namespace on the management cluster
$ kubectl apply -f cluster.yaml
```

aside of the default flavour, CAPV has the following:

- an `external-loadbalancer` flavour that enables you to to specify a pre-existing endpoint
- **DEPRECATED** an `haproxy` flavour to use HAProxy as a control plane endpoint

## Accessing the workload cluster

The kubeconfig for the workload cluster will be stored in a secret, which can
be retrieved using:

``` shell
$ kubectl get secret/vsphere-quickstart-kubeconfig -o json \
  | jq -r .data.value \
  | base64 --decode \
  > ./vsphere-quickstart.kubeconfig
```

The kubeconfig can then be used to apply a CNI for networking, for example, Calico:

```shell
KUBECONFIG=vsphere-quickstart.kubeconfig kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
```

after that you should see your nodes turn into ready:

```shell
$ KUBECONFIG=vsphere-quickstart.kubeconfig kubectl get nodes
NAME                                                          STATUS     ROLES    AGE   VERSION
vsphere-quickstart-9qtfd                                      Ready      master   47m   v1.17.3

```

## custom cluster templates

the provided cluster templates are quickstarts. If you need anything specific that requires a more complex setup, we recommand to use custom templates:

```shell
$ clusterctl config custom-cluster vsphere-quickstart \
    --infrastructure vsphere \
    --kubernetes-version v1.17.3 \
    --control-plane-machine-count 1 \
    --worker-machine-count 3 \
    --from ~/workspace/custom-cluster-template.yaml > custom-cluster.yaml
```

<!-- References -->
[vm-template]: https://docs.vmware.com/en/VMware-vSphere/6.7/com.vmware.vsphere.vm_admin.doc/GUID-17BEDA21-43F6-41F4-8FB2-E01D275FE9B4.html
[cluster-api-book]: https://cluster-api.sigs.k8s.io/
[glossary-bootstrapping]: https://cluster-api.sigs.k8s.io/reference/glossary.html#bootstrap
[kind]: https://kind.sigs.k8s.io
[glossary-management-cluster]: https://github.com/kubernetes-sigs/cluster-api/blob/master/docs/book/GLOSSARY.md#management-cluster
[releases]: https://github.com/kubernetes-sigs/cluster-api/releases
[docker]: https://docs.docker.com/glossary/?term=install
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[ovas]: ../README.md#kubernetes-versions-with-published-ovas
[default-machine-image]: https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova
[haproxy-machine-image]: https://storage.googleapis.com/capv-images/extra/haproxy/release/v0.6.4/capv-haproxy-v0.6.4.ova
[image-builder]: https://github.com/kubernetes-sigs/image-builder
[govc]: https://github.com/vmware/govmomi/tree/master/govc
 
