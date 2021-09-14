package installer

type ubuntu_20_4 struct {
	installer
}

func (u *ubuntu_20_4) Init() {
	u.addSwapOffStep("echo disabling swap Ubuntu 20.04")
	u.addSwapOnStep("echo enabling swap Ubuntu 20.04")
	u.addFirewallOffStep("echo disabling firewall Ubuntu 20.04")
	u.addFirewallOnStep("echo enabling firewall Ubuntu 20.04")

	u.addInstallStep("echo Ubuntu 20.04 install step")
	u.addUninstallStep("echo Ubuntu 20.04 uninstall step")
}

func (u *ubuntu_20_4) Install() {
	u.install()
}

func (u *ubuntu_20_4) Uninstall() {
	u.uninstall()
}

func (u *ubuntu_20_4) PrintSteps() {
	u.printSteps()
}
