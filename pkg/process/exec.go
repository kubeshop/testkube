package process

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Execute runs system command and returns whole output also in case of error
func Execute(command string, arguments ...string) (out []byte, err error) {
	return ExecuteInDir("", command, arguments...)
}

func ExecuteInDir(dir string, command string, arguments ...string) (out []byte, err error) {
	cmd := exec.Command(command, arguments...)
	if dir != "" {
		cmd.Dir = dir
	}
	buffer := new(bytes.Buffer)
	cmd.Stdout = buffer
	cmd.Stderr = buffer
	cmd.Start()
	err = cmd.Wait()
	if err != nil {
		// TODO clean error output (currently it has buffer too - need to refactor in cmd)
		return buffer.Bytes(), fmt.Errorf("process error: %w\noutput: %s", err, buffer.String())
	}

	return buffer.Bytes(), nil
}
