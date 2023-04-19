package commands

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/git"
	"github.com/kubeshop/testkube/pkg/helm"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/semver"
	"github.com/kubeshop/testkube/pkg/ui"
)

var appName string

func NewReleaseCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Release Helm Chart image",
		Long:  `Release Helm Chart, bump version, put version as helm app and chart version, create tag, push`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Verbose = verbose

			if dev {
				ui.Warn("Using prerelease version mode")
			} else {
				ui.Warn("Using production version mode")
			}

			currentAppVersion := getCurrentAppVersion()
			nextAppVersion := getNextVersion(dev, currentAppVersion, kind)
			pushVersionTag(nextAppVersion)

			// Let's checkout helm chart repo and put changes to particular app
			dir, err := git.PartialCheckout("https://github.com/kubeshop/helm-charts.git", "", appName, "main", "", "")
			ui.ExitOnError("checking out "+appName+" chart to "+dir, err)

			chart, path, err := helm.GetChart(dir)
			ui.ExitOnError("getting chart path", err)
			ui.Info("Current "+path+" version", helm.GetVersion(chart))
			valuesPath := strings.Replace(path, "Chart.yaml", "values.yaml", -1)

			// save version in Chart.yaml
			err = helm.SaveString(&chart, "version", nextAppVersion)
			ui.PrintOnError("Saving version string", err)

			err = helm.SaveString(&chart, "appVersion", nextAppVersion)
			ui.PrintOnError("Saving appVersion string", err)

			err = helm.UpdateValuesImageTag(valuesPath, nextAppVersion)
			ui.PrintOnError("Updating values image tag", err)

			err = helm.Write(path, chart)
			ui.ExitOnError("saving "+appName+" Chart.yaml file", err)

			gitAddCommitAndPush(dir, "updating "+appName+" chart version to "+nextAppVersion)

			// Checkout main testkube chart and bump main chart with next version
			dir, err = git.PartialCheckout("https://github.com/kubeshop/helm-charts.git", "", "testkube", "main", "", "")
			ui.ExitOnError("checking out testkube chart to "+dir, err)

			chart, path, err = helm.GetChart(dir)
			ui.ExitOnError("getting chart path", err)

			testkubeVersion := helm.GetVersion(chart)
			nextTestkubeVersion := getNextVersion(dev, testkubeVersion, semver.Patch)
			ui.Info("Generated new testkube version", nextTestkubeVersion)

			// bump main testkube chart version
			err = helm.SaveString(&chart, "version", nextTestkubeVersion)
			ui.PrintOnError("Saving version string", err)

			err = helm.SaveString(&chart, "appVersion", nextTestkubeVersion)
			ui.PrintOnError("Saving appVersion string", err)

			// set app dependency version
			_, err = helm.UpdateDependencyVersion(chart, appName, nextAppVersion)
			ui.PrintOnError("Updating dependency version", err)

			err = helm.Write(path, chart)
			ui.ExitOnError("saving testkube Chart.yaml file", err)

			gitAddCommitAndPush(dir, "updating testkube to "+nextTestkubeVersion+" and "+appName+" to "+nextAppVersion)

			tab := ui.NewArrayTable([][]string{
				{appName + " previous version", currentAppVersion},
				{"testkube previous version", testkubeVersion},
				{appName + " next version", nextAppVersion},
				{"testkube next version", nextTestkubeVersion},
			})

			ui.NL()
			ui.Table(tab, os.Stdout)

			ui.Completed("Release completed - Helm charts: ", "testkube:"+nextTestkubeVersion, appName+":"+nextAppVersion)
			ui.NL()
		},
	}

	cmd.Flags().StringVarP(&appName, "app", "a", "testkube-api", "app name chart")
	cmd.Flags().StringVarP(&kind, "kind", "k", "patch", "version kind one of (patch|minor|major")
	cmd.Flags().BoolVarP(&verbose, "verbose", "", false, "verbosity level")
	cmd.Flags().BoolVarP(&dev, "dev", "d", false, "generate beta increment")

	return cmd
}

func pushVersionTag(nextAppVersion string) {
	// set new tag and push
	// add "v" for go compatibility (Semver don't have it as prefix)
	_, err := process.Execute("git", "tag", "v"+nextAppVersion)
	ui.ExitOnError("tagging new version", err)

	_, err = process.Execute("git", "push", "--tags")
	ui.ExitOnError("pushing new version to repository", err)

}

func getCurrentAppVersion() string {
	// get current app version based on tags
	out, err := process.Execute("git", "tag")
	ui.ExitOnError("getting tags", err)

	versions := strings.Split(string(out), "\n")
	currentAppVersion := semver.GetNewest(versions)
	ui.Info("Current version based on tags", currentAppVersion)

	return currentAppVersion
}

func getNextVersion(dev bool, currentVersion string, kind string) (nextVersion string) {
	var err error
	switch true {
	case dev && semver.IsPrerelease(currentVersion):
		nextVersion, err = semver.NextPrerelease(currentVersion)
	case dev && !semver.IsPrerelease(currentVersion):
		nextVersion, err = semver.Next(currentVersion, semver.Patch)
		// semver sorting prerelease parts as strings
		nextVersion = nextVersion + "-beta001"
	default:
		nextVersion, err = semver.Next(currentVersion, kind)
	}
	ui.ExitOnError("getting next version for "+kind, err)

	return

}

func gitAddCommitAndPush(dir, message string) {
	_, err := process.ExecuteInDir(dir, "git", "add", "charts/")
	ui.ExitOnError("adding changes in charts directory (+"+dir+"+)", err)

	_, err = process.ExecuteInDir(dir, "git", "commit", "-m", message)
	ui.ExitOnError(message, err)

	_, err = process.ExecuteInDir(dir, "git", "push")
	ui.ExitOnError("pushing changes", err)
}
