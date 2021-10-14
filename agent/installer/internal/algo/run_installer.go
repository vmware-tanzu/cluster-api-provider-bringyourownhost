package algo

func RunInstaller(operation string, bundlePath string, i BaseK8sInstaller) {

	i.BundlePath = bundlePath
	i.LogBuilder = LogBuilder{}
	i.LogBuilder.Reset()

	switch operation {
	case "install":
		i.install()
	case "uninstall":
		i.uninstall()
	default:
		cmdLineHelp()
	}

}

func cmdLineHelp() {
	println("Please specify operation: install/uninstall")
}
