module github.com/vmware-tanzu/cluster-api-provider-byoh

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/cluster-api v0.3.11-0.20210524195020-fca52981fe0a
	sigs.k8s.io/controller-runtime v0.9.0-beta.5
)
