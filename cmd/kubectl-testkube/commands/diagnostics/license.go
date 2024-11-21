package diagnostics

import (
	"github.com/kubeshop/testkube/pkg/diagnostics"
	"github.com/kubeshop/testkube/pkg/diagnostics/loader"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators/license"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func RegisterLicenseValidators(cmd *cobra.Command, d diagnostics.Diagnostics) {

	namespace := cmd.Flag("namespace").Value.String()
	keyOverride := cmd.Flag("key-override").Value.String()
	fileOverride := cmd.Flag("file-override").Value.String()

	l, err := loader.GetLicenseConfig(namespace, "")
	ui.ExitOnError("loading license data", err)

	if keyOverride != "" {
		l.EnterpriseLicenseKey = keyOverride
	}
	if fileOverride != "" {
		l.EnterpriseLicenseFile = fileOverride
	}

	// License validator
	licenseGroup := d.AddValidatorGroup("license", l.EnterpriseLicenseKey)
	if l.EnterpriseOfflineActivation {
		licenseGroup.AddValidator(license.NewFileValidator())
		licenseGroup.AddValidator(license.NewOfflineLicenseKeyValidator())
		licenseGroup.AddValidator(license.NewOfflineLicenseValidator(l.EnterpriseLicenseKey, l.EnterpriseLicenseFile))
	} else {
		licenseGroup.AddValidator(license.NewOnlineLicenseKeyValidator())
		licenseGroup.AddValidator(license.NewKeygenShValidator())
	}
}

func NewLicenseCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "license",
		Aliases: []string{"lic", "l"},
		Short:   "Diagnose license errors",
		Run:     RunLicenseCheckFunc(),
	}

	cmd.Flags().StringP("key-override", "k", "", "Pass License key manually (we will not try to locate it automatically)")
	cmd.Flags().StringP("file-override", "f", "", "Pass License file manually (we will not try to locate it automatically)")

	return cmd
}

func RunLicenseCheckFunc() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		d := diagnostics.New()
		RegisterLicenseValidators(cmd, d)

		err := d.Run()
		ui.ExitOnError("Running validations", err)
		ui.NL(2)
	}
}
