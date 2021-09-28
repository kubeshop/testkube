package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
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
			ui.Verbose = verbose

			if dev {
				ui.Warn("Using prerelease version mode")
			} else {
				ui.Warn("Using production version mode")
			}

			currentAppVersion := getCurrentAppVersion()
			nextAppVersion := getNextVersion(dev, currentAppVersion, kind)
			pushVersionTag(nextAppVersion)

			if !dev {
				err := updateVersionInInstallScript("v" + nextAppVersion)
				ui.ExitOnError("updating install.sh script with version v"+nextAppVersion, err)
			}

			// Let's checkout helm chart repo and put changes to particular app
			dir, err := git.PartialCheckout("https://github.com/kubeshop/helm-charts.git", appName, "main")
			ui.ExitOnError("checking out "+appName+" chart to "+dir, err)

			chart, path, err := helm.GetChart(dir)
			ui.ExitOnError("getting chart path", err)
			ui.Info("Current "+path+" version", helm.GetVersion(chart))
			valuesPath := strings.Replace(path, "Chart.yaml", "values.yaml", -1)

			// save version in Chart.yaml
			helm.SaveString(&chart, "version", nextAppVersion)
			helm.SaveString(&chart, "appVersion", nextAppVersion)
			helm.UpdateValuesImageTag(valuesPath, nextAppVersion)

			err = helm.Write(path, chart)
			ui.ExitOnError("saving "+appName+" Chart.yaml file", err)

			saveChartChanges(dir, "updating "+appName+" chart version to "+nextAppVersion)

			// Checkout main kubtest chart and bump main chart with next version
			dir, err = git.PartialCheckout("https://github.com/kubeshop/helm-charts.git", "kubtest", "main")
			ui.ExitOnError("checking out kubtest chart to "+dir, err)

			chart, path, err = helm.GetChart(dir)
			ui.ExitOnError("getting chart path", err)

			kubtestVersion := helm.GetVersion(chart)
			nextKubtestVersion := getNextVersion(dev, kubtestVersion, version.Patch)
			ui.Info("Generated new kubtest version", nextKubtestVersion)

			// bump main kubtest chart version
			helm.SaveString(&chart, "version", nextKubtestVersion)
			helm.SaveString(&chart, "appVersion", nextKubtestVersion)

			// set app dependency version
			helm.UpdateDependencyVersion(chart, appName, nextAppVersion)

			err = helm.Write(path, chart)
			ui.ExitOnError("saving kubtest Chart.yaml file", err)

			saveChartChanges(dir, "updating kubtest to "+nextKubtestVersion+" and "+appName+" to "+nextAppVersion)

			tab := ui.NewArrayTable([][]string{
				{appName + " previous version", currentAppVersion},
				{"kubtest previous version", kubtestVersion},
				{appName + " next version", nextAppVersion},
				{"kubtest next version", nextKubtestVersion},
			})

			ui.NL()
			ui.Table(tab, os.Stdout)

			ui.Completed("Release completed", "kubtest:"+nextKubtestVersion, appName+":"+nextAppVersion)
			ui.NL()
		},
	}

	cmd.Flags().StringVarP(&appName, "app", "a", "api-server", "app name chart")
	cmd.Flags().StringVarP(&kind, "kind", "k", "patch", "version kind one of (patch|minor|major")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbosity level")
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
	currentAppVersion := version.GetNewest(versions)
	ui.Info("Current version based on tags", currentAppVersion)

	return currentAppVersion
}

func getNextVersion(dev bool, currentVersion string, kind string) (nextVersion string) {
	var err error
	switch true {
	case dev && version.IsPrerelease(currentVersion):
		nextVersion, err = version.NextPrerelease(currentVersion)
	case dev && !version.IsPrerelease(currentVersion):
		nextVersion, err = version.Next(currentVersion, version.Patch)
		// semver sorting prerelease parts as strings
		nextVersion = nextVersion + "-beta001"
	default:
		nextVersion, err = version.Next(currentVersion, kind)
	}
	ui.ExitOnError("getting next version for "+kind, err)

	return

}

func saveChartChanges(dir, message string) {
	_, err := process.ExecuteInDir(dir, "git", "add", "charts/")
	ui.ExitOnError("adding changes in charts directory (+"+dir+"+)", err)

	_, err = process.ExecuteInDir(dir, "git", "commit", "-m", message)
	ui.ExitOnError(message, err)

	_, err = process.ExecuteInDir(dir, "git", "push")
	ui.ExitOnError("pushing changes", err)
}

func updateVersionInInstallScript(version string) error {
	input, err := ioutil.ReadFile("install.sh")
	if err != nil {
		return err
	}

	r := regexp.MustCompile(`KUBTEST_VERSION=${KUBTEST_VERSION:-"[^"]+"}`)
	output := r.ReplaceAll(input, []byte(fmt.Sprintf(`KUBTEST_VERSION=${KUBTEST_VERSION:-"%s"}`, version)))

	return ioutil.WriteFile("install.sh", output, 0644)

}
