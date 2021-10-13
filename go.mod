module github.com/vmware-tanzu/cluster-api-provider-byoh

go 1.16

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v0.4.2

require (
	cloud.google.com/go v0.83.0 // indirect
	github.com/containerd/containerd v1.5.4 // indirect
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/go-logr/logr v1.1.0
	github.com/jackpal/gateway v1.0.7
	github.com/kube-vip/kube-vip v0.3.8
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.4.1
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/pkg/errors v0.9.1
	github.com/theupdateframework/notary v0.7.0 // indirect
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.21.3
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.20.0 // indirect
	k8s.io/kubectl v0.21.2
	k8s.io/utils v0.0.0-20210722164352-7f3ee0f31471
	sigs.k8s.io/cluster-api v0.4.2
	sigs.k8s.io/cluster-api/test v0.4.2
	sigs.k8s.io/controller-runtime v0.9.6
	sigs.k8s.io/yaml v1.3.0
)
