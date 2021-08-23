module github.com/vmware-tanzu/cluster-api-provider-byoh

go 1.16

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v0.4.0

require (
	github.com/containerd/containerd v1.5.4 // indirect
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/jackpal/gateway v1.0.7 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.4.1
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron v1.2.0 // indirect
	github.com/theupdateframework/notary v0.7.0 // indirect
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/cluster-api v0.4.0
	sigs.k8s.io/cluster-api/test v0.4.0
	sigs.k8s.io/controller-runtime v0.9.3
	sigs.k8s.io/yaml v1.2.0
)
