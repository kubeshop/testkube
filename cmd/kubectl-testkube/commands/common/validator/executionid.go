package validator

import (
	"errors"

	"github.com/spf13/cobra"
)

func ExecutionID(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("please pass execution ID as argument")
	}
	return nil
}
