package algo

import (
	"path/filepath"
	"strings"
)

type AptStep struct {
	ShellStep
}

func (a *AptStep) create(k BaseK8sInstaller, pkgFileName string) Step {
	pkgName := strings.Split(pkgFileName, ".")[0]
	pkgAbsolutePath := filepath.Join(k.BundlePath, pkgFileName)

	return &ShellStep{
		Desc:    pkgName,
		DoCmd:   "apt install -y '" + pkgAbsolutePath + "'",
		UndoCmd: "apt remove -y " + pkgName}
}
