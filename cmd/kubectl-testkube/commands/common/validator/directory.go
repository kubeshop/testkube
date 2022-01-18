package validator

import (
	"errors"

	"github.com/spf13/cobra"
)

func ManifestsDirectory(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("please pass directory with manifest files as argument")
	}
	return nil
}
