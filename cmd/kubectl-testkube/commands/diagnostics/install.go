package diagnostics

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/diagnostics"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators/deps"
	"github.com/kubeshop/testkube/pkg/ui"
)

func RegisterInstallValidators(_ *cobra.Command, d diagnostics.Diagnostics) {
	depsGroup := d.AddValidatorGroup("install.dependencies", nil)
	depsGroup.AddValidator(deps.NewKubectlDependencyValidator())
	depsGroup.AddValidator(deps.NewHelmDependencyValidator())
}

func NewInstallCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"ins", "i"},
		Short:   "Diagnose pre-installation dependencies",
		Run:     RunInstallCheckFunc(),
	}

	return cmd
}

func RunInstallCheckFunc() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		d := diagnostics.New()
		RegisterInstallValidators(cmd, d)

		err := d.Run()
		ui.ExitOnError("Running validations", err)
		ui.NL(2)
	}
}
