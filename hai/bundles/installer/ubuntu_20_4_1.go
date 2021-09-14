package installer

type ubuntu_20_4_1 struct {
	installer
	parent ubuntu_20_4
}

func (u *ubuntu_20_4_1) Init() {
	u.parent.Init()

	//example: overwrite swapOff & swapOn but keep the firewall steps
	u.parent.addSwapOffStep("echo disabling swap Ubuntu 20.04.1")
	u.parent.addSwapOnStep("echo enabling swap Ubuntu 20.0.1")

	u.addInstallStep("echo Ubuntu 20.04.1 install step")
	u.addUninstallStep("echo Ubuntu 20.04.1 uninstall step")
}

func (u *ubuntu_20_4_1) Install() {
	u.parent.Install()
	u.install()
}

func (u *ubuntu_20_4_1) Uninstall() {
	u.parent.Uninstall()
	u.uninstall()
}

func (u *ubuntu_20_4_1) PrintSteps() {
	u.parent.PrintSteps()
	u.installer.printSteps()
}
