package common

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// NewDataFromFlags read input data from stdin or '--file' parameter
func NewDataFromFlags(cmd *cobra.Command) (data *string, err error) {
	var hasFile bool
	if cmd.Flag("file").Changed {
		// get file content
		file := cmd.Flag("file").Value.String()
		if file != "" {
			fileContent, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("reading file "+file+" error: %w", err)
			}

			content := string(fileContent)
			data = &content
			hasFile = true
		}
	}

	if stat, _ := os.Stdin.Stat(); (stat.Mode()&os.ModeCharDevice) == 0 && !hasFile {
		stdinContent, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin error: %w", err)
		}

		if len(stdinContent) != 0 {
			content := string(stdinContent)
			data = &content
		}
	}

	return data, nil
}
