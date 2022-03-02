package validator

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"
)

func DNS1123Subdomain(cmd *cobra.Command, args []string) error {
	// TODO validate test name as valid kubernetes resource name
	// ISO subdomain name
	if len(args) < 1 {
		return errors.New("please pass valid test name")
	}

	name := args[0]

	errors := validation.IsDNS1123Subdomain(name)
	if len(errors) > 0 {
		return fmt.Errorf("invalid name errors: %v", errors)
	}

	return nil
}
