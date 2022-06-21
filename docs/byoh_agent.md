# BYOH Agent

BYOH agent is a binary that runs on the hosts and its responsibility include -  
1. Setup - Install kubernetes components on the host
2. Registration - Register the host to the cluster capacity pool
3. Bootstrap - Convert the host to a kubernetes node

## Usage of BYOH agent

Below flags are supported by the agent:-  
```
--downloadpath string 
```
File System path to keep the downloads (default `/var/lib/byoh/bundles`)  
<br />

```
--feature-gates map[String]Bool
```
A set of key-value pairs that describe feature gates for alpha/experimental features.  
<br />
```
--kubeconfig string           
```
Paths to a kubeconfig. Only required if out-of-cluster.  
<br />
```
--label labelFlags       
```
labels to attach to the ByoHost CR in the form `labelname=labelVal` Eg: `--label site=apac --label cores=2`  
<br />
```
--metricsbindaddress string
```
metricsbindaddress is the TCP address that the controller should bind to for serving prometheus metrics.It can be set to "0" to disable the metrics serving (default ":8080")  
<br />
```
--namespace string
```
Namespace in the management cluster where you would like to register this host (default "default")  
<br />
```
--skip-installation
```
If you want to skip installation of the kubernetes component binaries  
<br />
```
-v,--v Level
```
number for the log level verbosity  
<br />
```
--version
```
Print the version of the agent

## Installation of k8s components

The agent installs the kubernetes components like kubectl, kubeadm and kubelet that are required during node bootstrap. Users can own the installation of these components and skip the k8s installation by the agent using `--skip-installation` flag. 

### Bootstrapping a k8s node

The agent uses kubeadm `init/join/reset` under the hood  to bootstrap and reset a k8s node.

Kubeadm requires **root access** on the host to boostrap a k8s node. Refer [GitHub issue](https://github.com/kubernetes/kubeadm/issues/57) for the discussion. Since, BYOH agent uses kubeadm for node bootstrap, it also requires root access.

The agent writes/removes certain files on the local file system during kubeadm init/join/reset.

```/etc/kubernetes/pki/``` -  This directory contains all the certificates and keys used by kubeadm for running the cluster.

```/etc/kubernetes/manifests/``` - Contains static pod manifests used by kubelet during bootstrap.

```/etc/kubernetes/``` - Contains kubeconfig files with identities for control plane components.

```/run/kubeadm/``` - Contains configuration files kubeadm.yaml and kubeadm-join-config.yaml for kubeadm init/join.

```/run/cluster-api/``` - Contains a sentinel file `bootstrap-success.complete` created by bootstrap provider upon successful bootstrapping of a Kubernetes node. This allows infrastructure providers to detect and act on bootstrap failures.

```/var/lib/kubelet/``` - Contains configuration files for kubelet

```/var/lib/etcd/``` - It is the directory where etcd places its data.

The above directories contain files that are used for functioning of cluster (created as part of kubeadm init/join). The agent **does not** perform any OS level changes on the host.

BYOH agent also performs below operations to start/stop/check-status of certain processes.

```shell
systemctl is-active
```

```shell
systemd-resolved
```

```shell
systemctl status firewalld
```

```shell
systemctl is-active firewalld
```

```shell
uname (provides details about the current machine and its operating system)
```

```shell
systemctl status kubelet
```

```shell
systemctl stop kubelet
```

```shell
systemctl daemon-reload
```

```shell
systemctl restart kubelet
```