package validator

import (
	"errors"

	"github.com/spf13/cobra"
)

func TestSuiteName(cmd *cobra.Command, args []string) error {
	// TODO validate test name as valid kubernetes resource name

	if len(args) < 1 {
		return errors.New("please pass valid test suite name")
	}
	return nil
}
