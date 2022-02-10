package validator

import (
	"errors"

	"github.com/spf13/cobra"
)

func TestName(cmd *cobra.Command, args []string) error {
	// TODO validate test name as valid kubernetes resource name
	// ISO subdomain name

	if len(args) < 1 {
		return errors.New("please pass valid test name")
	}
	return nil
}
