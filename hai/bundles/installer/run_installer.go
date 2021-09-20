package installer

import "os"

func RunInstaller(osArgs []string, i Installer) {

	if len(osArgs) < 2 {
		cmdLineHelp()
		os.Exit(0)
	}

	switch osArgs[1] {
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
