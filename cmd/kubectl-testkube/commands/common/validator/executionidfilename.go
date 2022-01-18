package validator

import (
	"errors"

	"github.com/spf13/cobra"
)

func ExecutionIDAndFileNames(cmd *cobra.Command, args []string) error {
	if len(args) < 3 {
		return errors.New("please pass 'Execution ID' ,'Filename' and 'Destination Directory' as arguments")
	}
	return nil
}
