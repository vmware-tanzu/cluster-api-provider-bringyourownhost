package installer

type installer struct {
	installSteps   []step
	uninstallSteps []step
	swapOff        step
	swapOn         step
	firewallOff    step
	firewallOn     step
}

type NewInstaller interface {
	Install()
	Uninstall()
	Init()
	PrintSteps()
}

func (i *installer) install() {
	i.executeSteps(i.installSteps)
	i.swapOff.execute()
	i.firewallOff.execute()
}

func (i *installer) uninstall() {
	i.executeSteps(i.uninstallSteps)
	i.swapOn.execute()
	i.firewallOn.execute()
}

func (i *installer) executeSteps(steps []step) {
	for _, step := range steps {
		if len(step.cmd) > 0 {
			step.execute()
		}
	}
}

func (i *installer) addInstallStep(execCmd string) {
	var s step
	s.cmd = execCmd

	i.installSteps = append(i.installSteps, s)
}

func (i *installer) addUninstallStep(execCmd string) {
	var s step
	s.cmd = execCmd

	i.uninstallSteps = append(i.uninstallSteps, s)
}

func (i *installer) addSwapOffStep(execCmd string) {
	i.swapOff.cmd = execCmd
}

func (i *installer) addSwapOnStep(execCmd string) {
	i.swapOn.cmd = execCmd
}

func (i *installer) addFirewallOffStep(execCmd string) {
	i.firewallOff.cmd = execCmd
}

func (i *installer) addFirewallOnStep(execCmd string) {
	i.firewallOn.cmd = execCmd
}

func (i *installer) printSteps() {
	for _, step := range i.installSteps {
		println(step.cmd)
	}

	if len(i.swapOff.cmd) > 0 {
		println(i.swapOff.cmd)
	}

	if len(i.swapOn.cmd) > 0 {
		println(i.swapOn.cmd)
	}

	if len(i.firewallOff.cmd) > 0 {
		println(i.firewallOff.cmd)
	}

	if len(i.firewallOn.cmd) > 0 {
		println(i.firewallOn.cmd)
	}
}
