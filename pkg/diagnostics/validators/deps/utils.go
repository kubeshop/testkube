package deps

import (
	"os"
	"os/exec"
)

func checkFileExists(fileName string) bool {
	path, err := exec.LookPath(fileName)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return !os.IsNotExist(err)
}
