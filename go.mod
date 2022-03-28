module github.com/vmware-tanzu/cluster-api-provider-bringyourownhost

go 1.16

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.0.4

require (
	github.com/containerd/containerd v1.5.10 // indirect
	github.com/cppforlife/go-cli-ui v0.0.0-20200716203538-1e47f820817f
	github.com/docker/cli v20.10.13+incompatible
	github.com/docker/docker v20.10.12+incompatible
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/go-logr/logr v1.2.0
	github.com/jackpal/gateway v1.0.7
	github.com/k14s/imgpkg v0.18.0
	github.com/kube-vip/kube-vip v0.3.8
	github.com/maxbrunsfeld/counterfeiter/v6 v6.5.0
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/opencontainers/runc v1.0.3 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	github.com/theupdateframework/notary v0.7.0 // indirect
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	k8s.io/component-base v0.23.5
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.22.2
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	sigs.k8s.io/cluster-api v1.0.4
	sigs.k8s.io/cluster-api/test v1.0.4
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/yaml v1.3.0
)
