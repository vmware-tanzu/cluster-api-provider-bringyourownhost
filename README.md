# Kubernetes Cluster API Provider BYOH(Bring your own host)
<p align="center">
<!-- lint card --><a href="https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions/workflows/lint.yml" width="160x">
<img src="https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions/workflows/lint.yml/badge.svg" width="160x"></a>
<!-- test status -->
<a href="https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions/workflows/main.yml" width="160x">
<img src="https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions/workflows/main.yml/badge.svg" width="200x"></a>
<!-- go doc / reference card -->
<a href="https://pkg.go.dev/github.com/vmware-tanzu/cluster-api-provider-byoh"  width="160x">
<img src="https://pkg.go.dev/badge/github.com/vmware-tanzu/cluster-api-provider-byoh.svg"  width="100x"></a>
</p>

------

## What is Cluster API Provider BYOH

[Cluster API][cluster_api] brings
declarative, Kubernetes-style APIs to cluster creation, configuration and
management.

__BYOH__ is a Cluster API v1beta1 Infrastructure Provider for already-provisioned hosts running Linux.

## Getting Started
Check out the [getting_started](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/blob/main/docs/getting_started.md) guide for launching a BYOH workload cluster

## Project Status

This project is currently a work-in-progress, in an Alpha state, so it may not be production ready. There is no backwards-compatibility guarantee at this point. For more details on the roadmap and upcoming features, check out [the project's issue tracker on GitHub][issue].

## Features

- Native Kubernetes manifests and API
- Support for single and multi-node control plane clusters
- Support already provisioned Linux VMs with Ubuntu 20.04

## Getting involved and contributing

### Launching a Kubernetes cluster using BYOH source code

Check out the [developer guide](./docs/local_dev.md) for launching a BYOH cluster consisting of Docker containers as hosts.

More about development and contributing practices can be found in [`CONTRIBUTING.md`](./CONTRIBUTING.md).

------

## Compatibility with Cluster API and Kubernetes Versions

### Cluster API compatibility Matrix:

||Cluster API v1alpha4 (v0.4)|Cluster API v1beta1 (v1.0)|
|-|-|-|
|BYOH Provider v1alpha1 (v0.1.0)||✓|


### Kubernetes compatibility Matrix:

||Kubernetes 1.20|Kubernetes 1.21|Kubernetes 1.22|
|-|-|-|-|
|BYOH Provider v1alpha1 (v0.1.0)|||✓|




<!-- References -->

[cluster_api]: https://github.com/kubernetes-sigs/cluster-api
[issue]: https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/issues
