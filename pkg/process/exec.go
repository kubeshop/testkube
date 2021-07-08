package process

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Execute runs system command and returns whole output also in case of error
func Execute(command string, arguments ...string) (out []byte, err error) {
	proc := exec.Command(command, arguments...)
	buffer := new(bytes.Buffer)
	proc.Stdout = buffer
	proc.Stderr = buffer
	proc.Start()
	err = proc.Wait()
	if err != nil {
		return out, fmt.Errorf("process error: %w\noutput: %s", err, buffer.String())
	}

	return buffer.Bytes(), nil
}
