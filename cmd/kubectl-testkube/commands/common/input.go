package common

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// StdinPath is the path a file flag takes to read the command's standard input instead.
const StdinPath = "-"

// ReadFileOrStdin reads the file a flag names, or the command's standard input when it names
// StdinPath, so that content can be piped in: `cat payload.json | testkube create webhook
// --payload-template -`.
func ReadFileOrStdin(cmd *cobra.Command, path string) ([]byte, error) {
	if path == StdinPath {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(path)
}
