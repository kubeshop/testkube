package command

import (
	"bytes"
	"os/exec"
)

func Run(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	return out.Bytes(), err
}
