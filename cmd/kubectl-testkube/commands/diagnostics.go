package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"

	"github.com/kubeshop/testkube/pkg/diagnostics"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators/license"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators/mock"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDebugCmd creates the 'testkube debug' command
func NewDiagnosticsCmd() *cobra.Command {

	var validators common.CommaList
	var groups common.CommaList
	var key, file string

	cmd := &cobra.Command{
		Use:     "diagnostics",
		Aliases: []string{"diag", "di"},
		Short:   "Diagnoze testkube issues with ease",
		Run:     NewRunDiagnosticsCmdFunc(key, &validators, &groups),
	}

	allValidatorStr := ""
	allGroupsStr := ""

	cmd.Flags().VarP(&validators, "commands", "s", "Comma-separated list of validators: "+allValidatorStr+", defaults to all")
	cmd.Flags().VarP(&groups, "groups", "g", "Comma-separated list of groups, one of: "+allGroupsStr+", defaults to all")

	cmd.Flags().StringVarP(&key, "key", "k", "", "License key")
	cmd.Flags().StringVarP(&file, "file", "f", "", "License file")

	return cmd
}

func NewRunDiagnosticsCmdFunc(key string, commands, groups *common.CommaList) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {

		// Fetch current setup:
		offlineActivation := true
		key := cmd.Flag("key").Value.String()
		file := cmd.Flag("file").Value.String()

		// Compose diagnostics validators
		d := diagnostics.New()

		licenseKeyGroup := d.AddValidatorGroup("license.key", key)
		if offlineActivation {
			licenseKeyGroup.AddValidator(license.NewOfflineLicenseKeyValidator())
		} else {
			licenseKeyGroup.AddValidator(license.NewOnlineLicenseKeyValidator())
		}
		// common validator for both keys
		licenseKeyGroup.AddValidator(license.NewKeygenShValidator())
		licenseKeyGroup.AddValidator(mock.AlwaysValidValidator{Name: "Key presence"})
		licenseKeyGroup.AddValidator(mock.AlwaysInvalidMultiValidator{Name: "aaa1"})

		licenseFileGroup := d.AddValidatorGroup("license.file", file)
		licenseFileGroup.AddValidator(license.NewFileValidator())

		licenseFileGroup.AddValidator(mock.AlwaysValidValidator{Name: "Date occurance"})
		licenseFileGroup.AddValidator(mock.AlwaysValidValidator{Name: "Date range"})
		licenseFileGroup.AddValidator(mock.AlwaysInvalidMultiValidator{Name: "aaa1"})
		licenseFileGroup.AddValidator(mock.AlwaysInvalidValidator{Name: "aaa2"})

		// Run single "diagnostic"

		// Run multiple

		// Run predefined group

		// Run all
		err := d.Run()
		ui.ExitOnError("Running validations", err)

	}
}
