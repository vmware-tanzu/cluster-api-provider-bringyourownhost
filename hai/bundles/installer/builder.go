package installer

type Builder struct {
}

func (b *Builder) NewInstaller() BaseInstaller {

	var installer BaseInstaller

	os := "ubuntu_20_04_1_tkg_1_22" //temporary hardcoded ver

	switch os {
	case "ubuntu_20_04_1_tkg_1_22":
		installer = new(ubuntu_20_4_1_tkg_1_22)
	case "ubuntu_20_04_tkg_1_22":
		installer = new(ubuntu_20_4_tkg_1_22)
	default:
		installer = nil
	}

	installer.Init()
	installer.InitSteps()

	return installer
}
