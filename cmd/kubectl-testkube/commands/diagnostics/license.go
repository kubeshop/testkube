package diagnostics

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/diagnostics"
	"github.com/kubeshop/testkube/pkg/diagnostics/loader"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators/license"
	"github.com/kubeshop/testkube/pkg/ui"
)

func RegisterLicenseValidators(cmd *cobra.Command, d diagnostics.Diagnostics) {

	namespace := cmd.Flag("namespace").Value.String()
	keyOverride := cmd.Flag("key-override").Value.String()
	fileOverride := cmd.Flag("file-override").Value.String()
	isOfflineOverride := cmd.Flag("offline-override").Changed && cmd.Flag("offline-override").Value.String() == "true"

	// if not namespace provided load all namespaces having license file secret
	if !cmd.Flag("namespace").Changed {
		namespaces, err := common.KubectlGetNamespacesHavingSecrets("testkube-enterprise-license")
		if err != nil {
			ui.Errf("Can't check for namespaces, make sure you have valid access rights to list resources in Kubernetes")
			ui.ExitOnError("error:", err)
			return
		}

		switch true {
		case len(namespaces) == 0:
			ui.Failf("Can't locate any Testkube installations please pass `--namespace` parameter")
		case len(namespaces) == 1:
			namespace = namespaces[0]
		case len(namespaces) > 1:
			namespace = ui.Select("Choose namespace to check license", namespaces)
		}
	}

	var err error
	l := loader.License{}

	if keyOverride != "" {
		l.EnterpriseLicenseKey = keyOverride
	}
	if fileOverride != "" {
		l.EnterpriseLicenseFile = fileOverride
	}

	if fileOverride != "" && keyOverride != "" {
		l.EnterpriseOfflineActivation = true
	}
	if isOfflineOverride {
		l.EnterpriseOfflineActivation = isOfflineOverride
	}

	if keyOverride == "" || (l.EnterpriseOfflineActivation && fileOverride == "") {
		l, err = loader.GetLicenseConfig(namespace, "")
		ui.ExitOnError("loading license data", err)
	}

	// License validator
	if l.EnterpriseOfflineActivation {
		licenseGroup := d.AddValidatorGroup("offline.license.validation", l.EnterpriseLicenseKey)
		licenseGroup.AddValidator(license.NewFileValidator())
		licenseGroup.AddValidator(license.NewOfflineLicenseKeyValidator())
		licenseGroup.AddValidator(license.NewOfflineLicenseValidator(l.EnterpriseLicenseKey, l.EnterpriseLicenseFile))
	} else {
		licenseGroup := d.AddValidatorGroup("online.license.validation", l.EnterpriseLicenseKey)
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

	cmd.Flags().Bool("offline-override", false, "Pass License key manually (we will not try to locate it automatically)")
	cmd.Flags().StringP("key-override", "k", "", "Pass License key manually (we will not try to locate it automatically)")
	cmd.Flags().StringP("file-override", "f", "", "Pass License file manually (we will not try to locate it automatically)")

	return cmd
}

func RunLicenseCheckFunc() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		ui.H1("Check licensing issues")

		d := diagnostics.New()
		RegisterLicenseValidators(cmd, d)

		err := d.Run()
		ui.ExitOnError("Running validations", err)
		ui.NL(2)
	}
}
