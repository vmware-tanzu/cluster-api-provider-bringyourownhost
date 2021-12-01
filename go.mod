module github.com/vmware-tanzu/cluster-api-provider-bringyourownhost

go 1.16

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.0.0

require (
	github.com/containerd/containerd v1.5.8 // indirect
	github.com/cppforlife/go-cli-ui v0.0.0-20200716203538-1e47f820817f
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/jackpal/gateway v1.0.7
	github.com/k14s/imgpkg v0.18.0
	github.com/kube-vip/kube-vip v0.3.8
	github.com/maxbrunsfeld/counterfeiter/v6 v6.4.1
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/opencontainers/image-spec v1.0.2 //indirect
	github.com/pkg/errors v0.9.1
	github.com/theupdateframework/notary v0.7.0 // indirect
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.22.2
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/cluster-api v1.0.0
	sigs.k8s.io/cluster-api/test v1.0.0
	sigs.k8s.io/controller-runtime v0.10.2
	sigs.k8s.io/yaml v1.3.0
)
