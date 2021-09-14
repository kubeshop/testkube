package commands

import (
	"strings"

	"github.com/kubeshop/kubtest/pkg/git"
	"github.com/kubeshop/kubtest/pkg/helm"
	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/kubeshop/kubtest/pkg/version"
	"github.com/spf13/cobra"
)

var appName string

func NewReleaseCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Release Helm Chart image",
		Long:  `Release Helm Chart, bump version, put version as helm app and chart version, create tag, push`,
		Run: func(cmd *cobra.Command, args []string) {

			// get current version
			out, err := process.Execute("git", "tag")
			ui.ExitOnError("getting tags", err)

			versions := strings.Split(string(out), "\n")
			currentVersion := version.GetNewest(versions)
			ui.Info("Current version based on tags", currentVersion)

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

			// Let's checkout helm chart repo and put changes to particular app
			dir, err := git.PartialCheckout("https://github.com/kubeshop/helm-charts.git", appName, "main")
			ui.ExitOnError("checking out helm charts to "+dir, err)

			chart, path, err := helm.GetChart(dir)
			ui.ExitOnError("getting chart path", err)
			ui.Info("Current "+path+" version", helm.GetVersion(chart))
			valuesPath := strings.Replace(path, "Chart.yaml", "values.yaml", -1)

			// save version in Chart.yaml
			helm.SaveString(&chart, "version", nextVersion)
			helm.SaveString(&chart, "appVersion", nextVersion)
			helm.UpdateValuesImageTag(valuesPath, nextVersion)

			err = helm.Write(path, chart)
			ui.ExitOnError("saving "+appName+" Chart.yaml file", err)

			_, err = process.ExecuteInDir(dir, "git", "add", "charts/")
			ui.ExitOnError("adding changes in charts directory", err)

			_, err = process.ExecuteInDir(dir, "git", "commit", "-m", "updating api-server chart version to "+nextVersion)
			ui.ExitOnError("updating chart version to"+nextVersion, err)

			_, err = process.ExecuteInDir(dir, "git", "push")
			ui.ExitOnError("pushing changes", err)

			// Checkout main kubtest chart and bump main chart with next version
			dir, err = git.PartialCheckout("https://github.com/kubeshop/helm-charts.git", "kubtest", "main")
			ui.ExitOnError("checking out helm charts to "+dir, err)

			chart, path, err = helm.GetChart(dir)
			ui.ExitOnError("getting chart path", err)

			kubtestVersion := helm.GetVersion(chart)
			var nextKubtestVersion string
			switch true {
			case dev && version.IsPrerelease(kubtestVersion):
				nextKubtestVersion, err = version.NextPrerelease(kubtestVersion)
			case dev && !version.IsPrerelease(kubtestVersion):
				nextKubtestVersion, err = version.Next(kubtestVersion, version.Patch)
				nextKubtestVersion = nextKubtestVersion + "-beta1"
			default:
				nextKubtestVersion, err = version.Next(kubtestVersion, kind)
			}
			ui.ExitOnError("getting next version for kubtest ", err)
			ui.Info("Generated new kubtest version", nextKubtestVersion)

			// bump main kubtest chart version
			helm.SaveString(&chart, "version", nextKubtestVersion)
			helm.SaveString(&chart, "appVersion", nextKubtestVersion)

			// set app dependency version
			helm.UpdateDependencyVersion(chart, appName, nextVersion)

			err = helm.Write(path, chart)
			ui.ExitOnError("saving kubtest Chart.yaml file", err)

			ui.Warn(appName+" upgrade completed, version upgraded from "+kubtestVersion+" to ", nextVersion)
		},
	}

	cmd.Flags().StringVarP(&appName, "app", "a", "api-server", "app name chart")
	cmd.Flags().StringVarP(&kind, "kind", "k", "patch", "version kind one of (patch|minor|major")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbosity level")
	cmd.Flags().BoolVarP(&dev, "dev", "d", false, "generate beta increment")

	ui.Verbose = verbose

	return cmd
}
