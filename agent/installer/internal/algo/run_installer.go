package algo

func RunInstaller(operation string, i BaseK8sInstaller) {

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
