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
		// TODO clean error output (currently it has buffer too - need to refactor in cmd)
		return buffer.Bytes(), fmt.Errorf("process error: %w\noutput: %s", err, buffer.String())
	}

	return buffer.Bytes(), nil
}
