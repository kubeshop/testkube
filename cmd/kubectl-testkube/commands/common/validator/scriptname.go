package validator

import (
	"errors"

	"github.com/spf13/cobra"
)

func ScriptName(cmd *cobra.Command, args []string) error {
	// TODO validate script name as valid kubernetes resource name

	if len(args) < 1 {
		return errors.New("please pass valid script-name")
	}
	return nil
}
