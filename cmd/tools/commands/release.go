package commands

import (
	"strings"

	"github.com/kubeshop/kubtest/pkg/helm"
	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/kubeshop/kubtest/pkg/version"
	"github.com/spf13/cobra"
)

func NewReleaseCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Release Helm Chart image",
		Long:  `Release Helm Chart, bump version, put version as helm app and chart version, create tag, push`,
		Run: func(cmd *cobra.Command, args []string) {

			out, err := process.Execute("git", "tag")
			ui.ExitOnError("getting tags", err)

			versions := strings.Split(string(out), "\n")
			currentVersion := version.GetNewest(versions)
			ui.Info("Current version based on tags", currentVersion)

			chart, path, err := helm.GetChart("charts/")
			ui.ExitOnError("getting chart path", err)
			ui.Info("Current "+path+" version", helm.GetVersion(chart))
			valuesPath := strings.Replace(path, "Chart.yaml", "values.yaml", -1)

			// generate next version
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

			// add "v" for go compatibility (Semver don't have it as prefix)
			// set new tag and push
			_, err = process.Execute("git", "tag", "v"+nextVersion)
			ui.ExitOnError("tagging new version", err)

			_, err = process.Execute("git", "push", "--tags")
			ui.ExitOnError("pushing new version to repository", err)

			// save version in Chart.yaml
			helm.SaveString(&chart, "version", nextVersion)
			helm.SaveString(&chart, "appVersion", nextVersion)
			helm.UpdateValuesImageTag(valuesPath, nextVersion)

			err = helm.Write(path, chart)
			ui.ExitOnError("saving Chart.yaml file", err)

			// save Chart.yaml, and push changes to git
			// as https://github.com/helm/chart-releaser-action/issues/60
			// we need to push changes after tag is created
			_, err = process.Execute("git", "add", "charts/")
			ui.ExitOnError("adding changes in charts directory", err)

			_, err = process.Execute("git", "commit", "-m", "updating chart version to "+nextVersion)
			ui.ExitOnError("updating chart version to"+nextVersion, err)

			_, err = process.Execute("git", "push")
			ui.ExitOnError("pushing changes", err)

			ui.Warn("Upgrade completed, version upgraded from "+currentVersion+" to ", nextVersion)
		},
	}

	cmd.Flags().StringVarP(&kind, "kind", "k", "patch", "version kind one of (patch|minor|major")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbosity level")
	cmd.Flags().BoolVarP(&dev, "dev", "d", false, "generate beta increment")

	ui.Verbose = verbose

	return cmd
}
