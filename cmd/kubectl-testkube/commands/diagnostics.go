package commands

import (
	"github.com/spf13/cobra"

	commands "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/diagnostics"
	"github.com/kubeshop/testkube/pkg/diagnostics"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDebugCmd creates the 'testkube debug' command
func NewDiagnosticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "diagnostics",
		Aliases: []string{"diagnose", "diag", "di"},
		Short:   "Diagnoze testkube issues with ease",
		Run:     NewRunDiagnosticsCmdFunc(),
	}

	cmd.Flags().Bool("offline-override", false, "Pass License key manually (we will not try to locate it automatically)")
	cmd.Flags().StringP("key-override", "k", "", "Pass License key manually (we will not try to locate it automatically)")
	cmd.Flags().StringP("file-override", "f", "", "Pass License file manually (we will not try to locate it automatically)")

	cmd.AddCommand(commands.NewLicenseCheckCmd())
	cmd.AddCommand(commands.NewInstallCheckCmd())

	return cmd
}

func NewRunDiagnosticsCmdFunc() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		d := diagnostics.New()

		commands.RegisterInstallValidators(cmd, d)
		commands.RegisterLicenseValidators(cmd, d)

		err := d.Run()
		ui.ExitOnError("Running validations", err)
		ui.NL(2)
	}
}
