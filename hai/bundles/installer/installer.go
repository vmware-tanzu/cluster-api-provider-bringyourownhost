package installer

type installer struct {
	steps []step
}

type BaseInstaller interface {
	Install()
	Uninstall()
	Init()
	InitSteps()
	CreateSwapStep() step
	CreateFirewallStep() step
	AddSteps([]step)
}

func (i *installer) Install() {
	for _, step := range i.GetSteps() {
		step.Execute()
	}
}

func (i *installer) Uninstall() {
	for _, step := range i.GetSteps() {
		step.Undo()
	}
}

func (i *installer) CreateStep(execCmd string, undoCmd string) step {
	var s step
	s.cmd = execCmd
	s.undo = undoCmd

	return s
}

func (i *installer) AddSteps(newSteps []step) {
	i.steps = append(i.steps, newSteps...)
}

func (i *installer) GetSteps() []step {
	return i.steps
}
