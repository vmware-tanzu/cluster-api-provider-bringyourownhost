package hainstaller

type StepBackup struct {
	Step
	Item string
}

func (bak StepBackup) Install() {
	println("backing up: " + bak.Item)
}

func (bak StepBackup) Uninstall() {
	println("restoring: " + bak.Item)
}
