package commands

import (
	"strings"

	"github.com/kubeshop/kubtest/pkg/helm"
	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/kubeshop/kubtest/pkg/version"
	"github.com/spf13/cobra"
)

func NewHelmReleaseCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "helm-release",
		Short: "Release Helm chart image",
		Long:  `Release Helm chart, bump version, put version as helm app and chart version, create tag, push`,
		Run: func(cmd *cobra.Command, args []string) {

			out, err := process.Execute("git", "tag")
			ui.ExitOnError("getting tags", err)

			versions := strings.Split(string(out), "\n")
			currentVersion := version.GetNewest(versions)
			ui.Info("Current version based on tags", currentVersion)

			// update chart
			chart, path, err := helm.GetChart("charts/")
			ui.ExitOnError("getting chart path", err)
			ui.Info("Current "+path+" version", helm.GetVersion(chart))

			var nextVersion string

			switch true {
			case dev && version.IsPrerelease(currentVersion):
				nextVersion, err = version.NextPrerelease(currentVersion)
			case dev && !version.IsPrerelease(currentVersion):
				nextVersion, err = version.Next(currentVersion, version.Patch)
				nextVersion = nextVersion + "-beta1"
			default:
				nextVersion, err = version.Next(currentVersion, kind)
			}
			ui.ExitOnError("getting next version for "+kind, err)
			ui.Warn("Upgrading version from "+currentVersion+" to:", nextVersion)

			helm.SaveString(&chart, "version", nextVersion)
			helm.SaveString(&chart, "appVersion", nextVersion)

			err = helm.Write(path, chart)
			ui.ExitOnError("saving Chart.yaml file", err)

			// add "v" for go compatibility (Semver don't have it as prefix)
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
	cmd.Flags().BoolVarP(&dev, "dev", "d", false, "generate beta increment")

	ui.Verbose = verbose

	return cmd
}
