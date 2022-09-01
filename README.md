# Kubernetes Cluster API Provider Bring Your Own Host (BYOH)
<p align="center">
<!-- lint card --><a href="https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions/workflows/lint.yml">
<img src="https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions/workflows/lint.yml/badge.svg"></a>
<!-- test status -->
<a href="https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions?query=event%3Apush+branch%3Amain+workflow%3ACI+">
<img src="https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions/workflows/ci.yml/badge.svg?branch=main&event=push"></a>
<!-- go doc / reference card -->
<a href="https://pkg.go.dev/github.com/vmware-tanzu/cluster-api-provider-bringyourownhost">
<img src="https://pkg.go.dev/badge/github.com/vmware-tanzu/cluster-api-provider-bringyourownhost.svg"></a>
<!-- goreportcard badge -->
<a href="https://goreportcard.com/report/github.com/vmware-tanzu/cluster-api-provider-bringyourownhost">
<img src="https://goreportcard.com/badge/github.com/vmware-tanzu/cluster-api-provider-bringyourownhost"></a>
<!-- codecov badge -->
<a href="https://codecov.io/gh/vmware-tanzu/cluster-api-provider-bringyourownhost">
<img src="https://codecov.io/gh/vmware-tanzu/cluster-api-provider-bringyourownhost/branch/main/graph/badge.svg?token=8GGPY0MENQ"></a>
<!-- openssf badge -->
<a href="https://bestpractices.coreinfrastructure.org/projects/5506">
<img src="https://bestpractices.coreinfrastructure.org/projects/5506/badge"></a>
</p>

------

## What is Cluster API Provider BYOH

[Cluster API](https://github.com/kubernetes-sigs/cluster-api) brings
declarative, Kubernetes-style APIs to cluster creation, configuration and
management.

__BYOH__ is a Cluster API Infrastructure Provider for already-provisioned hosts running Linux. This provider allows operators to adopt Cluster API for deploying and managing kubernetes nodes without also having to adopt a specific infrastructure service. This enables users to decouple kubernetes node provisioning from host and infrastructure provisioning.

## BYOH Glossary
**Host** - A host is a running computer system. It could be physical or virtual. It has a kernel and some base operating system

**BYO Host** - A Linux host provisioned and managed outside of Cluster API

**BYOH Capacity Pool** - A set of BYO Hosts registered in a management cluster & authorized for usage as a capacity for deploying Kubernetes nodes

**Kubernetes Node** - A Kubernetes Node that runs on top of a Host. There is a 1-to-1 relationship between nodes and hosts (every host has zero or one nodes). Node provisioning and lifecycle management is a Cluster API responsibility

**Kubernetes Host Components** - The components that run uncontainerized on the host and are required to bootstrap a Kubernetes node. Typically, this is at least kubelet, containerd and kubeadm, but different OS might require different components in this category

## Features

- Native Kubernetes manifests and API
- Support for single and multi-node control plane clusters
- Support already provisioned Linux VMs with Ubuntu 20.04

## Getting Started
Check out the [getting_started](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/blob/main/docs/getting_started.md) guide for launching a BYOH workload cluster

## Community, discussion, contribution, and support

The BringYourOwnHost provider is developed in the open, and is constantly being improved by our users, contributors, and maintainers.
If you have questions or want to get the latest project news, you can connect with us in the following ways:

- Chat with us on the Kubernetes [Slack](http://slack.k8s.io/) in the [#cluster-api](https://kubernetes.slack.com/archives/C8TSNPY4T) channel
- Subscribe to the [SIG Cluster Lifecycle](https://groups.google.com/forum/#!forum/kubernetes-sig-cluster-lifecycle) Google Group for access to documents and calendars
- Join our Cluster API Provider for BringYourOwnHost working group sessions where we share the latest project news, demos, answer questions, and triage issues
    - Weekly on Wednesdays @ 1:30PM Indian Standard Time on [Zoom](https://VMware.zoom.us/j/94476574480?pwd=WGYzOXBoL1VsVnBXK3c5TWd1bG5SZz09) - [convert to your timezone](https://dateful.com/time-zone-converter?t=13:30&tz=IST)
    - Previous meetings: \[ [notes](https://docs.google.com/document/d/1T-3_eskC_HCtXLh3PA8y--mgO-AIajZfevcnYuno6JM/edit#heading=h.y186zgz0eh6e) | [recordings](https://www.youtube.com/playlist?list=PLHbHoGHbooH41L5P-tIK6QqhILdEI9yBK) \]

Pull Requests and feedback on issues are very welcome!
See the [issue tracker](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/issues) if you're unsure where to start, especially the [Good first issue](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) and [Help wanted](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) tags, and
also feel free to reach out to discuss.

See also our [contributor guide](CONTRIBUTING.md) and the Kubernetes [community page](https://kubernetes.io/community) for more details on how to get involved.


## Project Status

This project is currently a work-in-progress, in an Alpha state, so it may not be production ready. There is no backwards-compatibility guarantee at this point. For more details on the roadmap and upcoming features, check out the project's [issue](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/issues) tracker on GitHub.


## Getting involved and contributing

### Launching a Kubernetes cluster using BYOH source code

Check out the [developer guide](./docs/local_dev.md) for launching a BYOH cluster consisting of Docker containers as hosts.

More about development and contributing practices can be found in [`CONTRIBUTING.md`](./CONTRIBUTING.md).

## Implement Custom Installer controller
An installer controller is responsible to provide the installation and uninstallation scripts for k8s dependencies, prerequisites and components on each `BYOHost`.  
If someone wants to implement their own installer controller then they need to follow the contract defined in [installer](./docs/installer.md) doc.

------

## Compatibility with Cluster API

- BYOH is currently compatible wth Cluster API v1beta1 (v1.0)

## Supported OS and Kubernetes versions
| Operating System  | Architecture  | Kubernetes v1.21.* | Kubernetes v1.22.* | Kubernetes v1.23.* |
| ------------------|---------------| :----------------: | :----------------: | :----------------: |
| Ubuntu 20.04.*    | amd64         |        ✓           |        ✓           |        ✓           |

**NOTE:**  The '*' in OS means that all Ubuntu 20.04 patches are supported.

**NOTE:**  The '*' in the K8s version means that the K8s minor release is supported but it may happen that a BYOH bundle for a specific patch may not exist in the OCI registry.

## BYOH in News
- [TGIK episode on BYOH](https://www.youtube.com/watch?v=Xwm5Ka27-Io&t=2838s)
- BYOH presented during [Cluster API Office Hours](https://www.youtube.com/watch?v=6ODMLgX-dz4&t=572s)
- [BYOH on ARM](https://williamlam.com/2021/11/hybrid-x86-and-arm-kubernetes-clusters-using-tanzu-community-edition-tce-and-esxi-arm.html)

