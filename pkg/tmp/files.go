package tmp

import (
	"os"
)

// Name generate new temp file and returns file path
func Name() string {
	tmpfile, _ := os.CreateTemp("", "testkube-tmp")
	return tmpfile.Name()
}
