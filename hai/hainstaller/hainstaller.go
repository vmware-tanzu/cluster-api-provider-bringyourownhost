package hainstaller

type Step interface {
	Install()
	Uninstall()
}

type Bundle struct {
	steps []Step
}

func (b *Bundle) AddStep(s Step) {
	b.steps = append(b.steps, s)
}

func (b *Bundle) Install(bundlePath string) {
	println("Processing bundle: " + bundlePath)

	for _, step := range b.steps {
		step.Install()
	}

	println("complete")
}

func (b *Bundle) Uninstall(bundlePath string) {
	println("Uninstalling bundle: " + bundlePath)

	for i := len(b.steps) - 1; i >= 0; i-- {
		b.steps[i].Uninstall()
	}

	println("complete")
}
