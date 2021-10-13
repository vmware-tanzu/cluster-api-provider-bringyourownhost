package installer

type osk8sInstaller interface{}
type k8sInstallerMap map[string]osk8sInstaller
type osk8sInstallerMap map[string]k8sInstallerMap

// Registry associates a (OS,K8sVersion) pair with an installer
type registry struct {
	osk8sInstallerMap
}

func NewRegistry() registry {
	return registry{make(osk8sInstallerMap)}
}

func (r *registry) Add(os, k8sVer string, installer osk8sInstaller) {
	if _, ok := r.osk8sInstallerMap[os]; !ok {
		r.osk8sInstallerMap[os] = make(k8sInstallerMap)
	}
	r.osk8sInstallerMap[os][k8sVer] = installer
}

func (r *registry) ListOs() []string {
	result := make([]string, 0, len(r.osk8sInstallerMap))
	for os := range(r.osk8sInstallerMap) {
		result = append(result, os)
	}
	return result
}

func (r *registry) ListK8s(os string) []string {
	result := make([]string, 0, len(r.osk8sInstallerMap[os]))
	for k8s := range(r.osk8sInstallerMap[os]) {
		result = append(result, k8s)
	}
	return result
}

func (r *registry) GetInstaller(os, k8sVer string) osk8sInstaller {
	return r.osk8sInstallerMap[os][k8sVer]
}
