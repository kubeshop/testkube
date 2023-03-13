package validator

import (
	"errors"

	"github.com/spf13/cobra"
)

func ExecutionName(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("please pass execution name as argument")
	}
	return nil
}
