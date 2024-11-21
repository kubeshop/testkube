package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"

	"github.com/kubeshop/testkube/pkg/diagnostics"
	"github.com/kubeshop/testkube/pkg/diagnostics/loader"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators/deps"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators/license"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDebugCmd creates the 'testkube debug' command
func NewDiagnosticsCmd() *cobra.Command {

	var validators common.CommaList
	var groups common.CommaList
	var key, file string

	cmd := &cobra.Command{
		Use:     "diagnostics",
		Aliases: []string{"diagnose", "diag", "di"},
		Short:   "Diagnoze testkube issues with ease",
		Run:     NewRunDiagnosticsCmdFunc(key, &validators, &groups),
	}

	allValidatorStr := ""
	allGroupsStr := ""

	cmd.Flags().VarP(&validators, "commands", "s", "Comma-separated list of validators: "+allValidatorStr+", defaults to all")
	cmd.Flags().VarP(&groups, "groups", "g", "Comma-separated list of groups, one of: "+allGroupsStr+", defaults to all")

	cmd.Flags().StringVarP(&key, "key-override", "k", "", "Pass License key manually (we will not try to locate it automatically)")
	cmd.Flags().StringVarP(&file, "file-override", "f", "", "Pass License file manually (we will not try to locate it automatically)")

	return cmd
}

func NewRunDiagnosticsCmdFunc(key string, commands, groups *common.CommaList) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {

		// Fetch current setup:
		namespace := cmd.Flag("namespace").Value.String()

		keyOverride := cmd.Flag("key-override").Value.String()
		fileOverride := cmd.Flag("file-override").Value.String()

		l, err := loader.GetLicenseConfig(namespace)
		ui.ExitOnError("loading license data", err)

		if keyOverride != "" {
			l.EnterpriseLicenseKey = keyOverride
		}
		if fileOverride != "" {
			l.EnterpriseLicenseFile = fileOverride
		}

		// Compose diagnostics validators
		d := diagnostics.New()

		depsGroup := d.AddValidatorGroup("install.dependencies", nil)
		depsGroup.AddValidator(deps.NewKubectlDependencyValidator())
		depsGroup.AddValidator(deps.NewHelmDependencyValidator())

		// License validator
		licenseKeyGroup := d.AddValidatorGroup("license.key", key)
		if l.EnterpriseOfflineActivation {
			licenseKeyGroup.AddValidator(license.NewOfflineLicenseKeyValidator())

			// for offline license also add license file validator
			licenseFileGroup := d.AddValidatorGroup("license.file", l.EnterpriseLicenseFile)
			licenseFileGroup.AddValidator(license.NewFileValidator())

			offlineLicenseGroup := d.AddValidatorGroup("license.offline.check", l.EnterpriseLicenseFile)
			offlineLicenseGroup.AddValidator(license.NewOfflineLicenseValidator(l.EnterpriseLicenseKey, l.EnterpriseLicenseFile))
		} else {
			licenseKeyGroup.AddValidator(license.NewOnlineLicenseKeyValidator())
		}

		// common validator for both key types
		licenseKeyGroup.AddValidator(license.NewKeygenShValidator())

		// TODO allow to run partially

		// Run single "diagnostic"

		// Run multiple

		// Run predefined group

		// Run all
		err = d.Run()
		ui.ExitOnError("Running validations", err)

		ui.NL(2)
	}
}
