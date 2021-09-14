package installer

type Builder struct {
}

func (b *Builder) NewInstaller(os string) NewInstaller {

	var installer NewInstaller

	switch os {
	case "ubuntu_20_04_1":
		installer = new(ubuntu_20_4_1)
	case "ubuntu_20_04":
		installer = new(ubuntu_20_4)
	default:
		installer = nil
	}

	return installer
}
