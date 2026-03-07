package tmp

import (
	"os"
)

// Name generate new temp file and returns file path
func Name() string {
	tmpfile, _ := os.CreateTemp("", "testkube-tmp")
	_ = tmpfile.Close()
	return tmpfile.Name()
}
