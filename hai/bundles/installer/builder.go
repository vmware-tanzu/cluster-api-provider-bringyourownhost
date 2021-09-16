package installer

type Builder struct {
}

func (b *Builder) NewInstaller() BaseInstaller {

	var installer BaseInstaller

	//TODO: replace this one with an appropriate algo
	//that detects the appropriate {OS/Distro, TKG Ver} pair
	os := "ubuntu_20_04_1_pkg_1_22"

	switch os {
	case "ubuntu_20_04_1_pkg_1_22":
		installer = new(ubuntu_20_4_1_pkg_1_22)
	case "ubuntu_20_04_pkg_1_22":
		installer = new(ubuntu_20_4_pkg_1_22)
	default:
		installer = nil
	}

	installer.Init()
	installer.InitSteps()

	return installer
}
