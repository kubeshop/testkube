package commands

import (
	"strings"

	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/kubeshop/kubtest/pkg/version"
	"github.com/spf13/cobra"
)

var verbose bool
var kind string

func NewVersionBumpCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "bump",
		Short: "Shows version and build info",
		Long:  `Shows version and build info`,
		Run: func(cmd *cobra.Command, args []string) {
			out, err := process.Execute("git", "tag")
			ui.ExitOnError("getting tags", err)

			versions := strings.Split(string(out), "\n")
			currentVersion := version.GetNewest(versions)
			nextVersion, err := version.Next(currentVersion, kind)
			ui.ExitOnError("getting next version for "+kind, err)
			nextVersion = "v" + nextVersion

			ui.Info("Generated new version", nextVersion)

			_, err = process.Execute("git", "tag", nextVersion)
			ui.ExitOnError("tagging new version", err)

			_, err = process.Execute("git", "push", "--tags")
			ui.ExitOnError("pushing new version to repository", err)

		},
	}

	cmd.Flags().StringVarP(&kind, "kind", "k", "patch", "version kind one of (patch|minor|major")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbosity level")

	ui.Verbose = verbose

	return cmd
}
