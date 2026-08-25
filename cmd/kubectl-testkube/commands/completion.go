package commands

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// pluginBinaryName is the actual name the binary is invoked under (it ships
// as a kubectl plugin, e.g. "kubectl-testkube"), as opposed to RootCmd.Use
// ("testkube"), which is only used for help text.
const pluginBinaryName = "kubectl-testkube"

// NewCompletionCmd builds a "completion" command that mirrors Cobra's
// built-in one for bash/fish/powershell, but generates the zsh script under
// the actual invoked binary name instead of RootCmd.Use.
//
// Cobra's generated zsh script hardcodes RootCmd.Name() (the first word of
// Use, i.e. "testkube") in its "#compdef" header and function names. Since
// this binary is always invoked as "kubectl-testkube" (a kubectl plugin),
// the installed script never matched the actual command name and zsh
// completion silently did nothing. See #964.
func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate the autocompletion script for the specified shell",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return genZshCompletionAs(root, pluginBinaryName, os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}

// genZshCompletionAs generates the zsh completion script as if the root
// command's Use were `name`, without permanently changing it (which would
// also affect --help output and other shells' generated scripts).
func genZshCompletionAs(root *cobra.Command, name string, w io.Writer) error {
	original := root.Use
	root.Use = name
	defer func() { root.Use = original }()
	return root.GenZshCompletion(w)
}
