package installer

type ubuntu_20_4_1_pkg_1_22 struct {
	ubuntu_20_4_pkg_1_22
}

func (u *ubuntu_20_4_1_pkg_1_22) Init() {
	u.AddSteps([]step{
		u.CreateSwapStep(),      //ubuntu_20_4_1 override
		u.CreateFirewallStep()}) //ubuntu_20_4
}

func (u *ubuntu_20_4_1_pkg_1_22) InitSteps() {
	u.ubuntu_20_4_pkg_1_22.InitSteps()

	u.AddSteps([]step{
		u.CreateStep(
			"echo Ubuntu 20.04.1 install step 1",
			"echo Ubuntu 20.04.1 uninstall step 1"),

		u.CreateStep(
			"echo Ubuntu 20.04.1 install step 2",
			"echo Ubuntu 20.04.1 uninstall step 2")})
}

func (u *ubuntu_20_4_1_pkg_1_22) CreateSwapStep() step {
	return u.CreateStep(
		"echo disabling swap Ubuntu 20.04.1",
		"echo enabling swap Ubuntu 20.04.1")
}
