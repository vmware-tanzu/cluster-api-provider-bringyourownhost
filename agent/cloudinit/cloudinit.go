package cloudinit

import (
	"fmt"
	"os/exec"
	"strings"
)

type ScriptExecutor struct {
}

func (se ScriptExecutor) Execute(bootstrapScript string) error {
	commands := strings.Split(bootstrapScript, " ")

	cmd := exec.Command(commands[0], strings.Join(commands[1:], " "))
	out, err := cmd.Output()

	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
