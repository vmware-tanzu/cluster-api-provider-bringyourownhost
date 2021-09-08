# Kubernetes Cluster API Provider Byoh(Bring your own host)
[![lint](https://github.com/vmware-tanzu/cluster-api-provider-byoh/actions/workflows/lint.yml/badge.svg)](https://github.com/vmware-tanzu/cluster-api-provider-byoh/actions/workflows/lint.yml)
[![e2e-test](https://github.com/vmware-tanzu/cluster-api-provider-byoh/actions/workflows/e2e.yml/badge.svg)](https://github.com/vmware-tanzu/cluster-api-provider-byoh/actions/workflows/e2e.yml)

------

## What is the Cluster API Provider Byoh

The [Cluster API][cluster_api] brings
declarative, Kubernetes-style APIs to cluster creation, configuration and
management.

__Byoh__ is a Cluster API v1alpha4 Infrastructure Provider for already-provisioned hosts running Linux.

## Project Status

This project is currently a work-in-progress, in an Alpha state, so it may not be production ready. There is no backwards-compatibility guarantee at this point. For more details on the roadmap and upcoming features, check out [the project's issue tracker on GitHub][issue].

## Launching a Kubernetes cluster using Byoh

Check out the [test guide](./test/test-run.md) for launching a cluster on Docker.

## Features

- Native Kubernetes manifests and API
- Support for single and multi-node control plane clusters
- Support already provisioned Linux VMs with Ubuntu 18.04 and 20.04

------

## Compatibility with Cluster API and Kubernetes Versions

### Cluster API compatibility Matrix:

||Cluster API v1alpha3 (v0.3)|Cluster API v1alpha4 (v0.4)|
|-|-|-|
|Byoh Provider v1alpha1 (v0.1.0)||✓|


### Kubernetes compatibility Matrix:

||Kubernetes 1.20|Kubernetes 1.21|Kubernetes 1.22|
|-|-|-|-|
|Byoh Provider v1alpha1 (v0.1.0)|||✓|


## Getting involved and contributing

More about development and contributing practices can be found in [`CONTRIBUTING.md`](./CONTRIBUTING.md).

<!-- References -->

[cluster_api]: https://github.com/kubernetes-sigs/cluster-api
[issue]: https://github.com/vmware-tanzu/cluster-api-provider-byoh/issues