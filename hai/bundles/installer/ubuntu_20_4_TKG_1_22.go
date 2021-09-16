package installer

type ubuntu_20_4_tkg_1_22 struct {
	installer
}

func (u *ubuntu_20_4_tkg_1_22) Init() {
	u.AddSteps([]step{
		u.CreateSwapStep(),
		u.CreateFirewallStep()})
}

func (u *ubuntu_20_4_tkg_1_22) InitSteps() {
	u.AddSteps([]step{
		u.CreateStep(
			"echo Ubuntu 20.04 install step 1",
			"echo Ubuntu 20.04 uninstall step 1"),

		u.CreateStep(
			"echo Ubuntu 20.04 install step 2",
			"echo Ubuntu 20.04 uninstall step 2")})
}

func (u *ubuntu_20_4_tkg_1_22) CreateSwapStep() step {
	return u.CreateStep(
		"echo disabling swap Ubuntu 20.04",
		"echo enabling swap Ubuntu 20.04")
}

func (u *ubuntu_20_4_tkg_1_22) CreateFirewallStep() step {
	return u.CreateStep(
		"echo disabling firewall Ubuntu 20.04",
		"echo enabling firewall Ubuntu 20.04")
}
