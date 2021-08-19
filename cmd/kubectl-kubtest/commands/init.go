package commands

import (
	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {

	RootCmd.AddCommand(installCmd)

	installCmd.Flags().String("chart", "kubtest/kubtest", "chart name")
	installCmd.Flags().String("name", "kubtest", "installation name")
	installCmd.Flags().String("namespace", "default", "namespace where to install")

	RootCmd.AddCommand(versionCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Helm chart registry in current kubectl context",
	Long:  `Install can be configured with use of particular `,
	Run: func(cmd *cobra.Command, args []string) {

		chart := cmd.Flag("chart").Value.String()
		name := cmd.Flag("name").Value.String()
		namespace := cmd.Flag("namespace").Value.String()

		_, err := process.Execute("helm", "repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts")
		ui.ExitOnError("adding kubtest repo", err)

		out, err := process.Execute("helm", "install", "--namespace", namespace, name, chart)
		ui.ExitOnError("executing helm install", err)

		ui.Info("Helm output", string(out))
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows version and build info",
	Long:  `Shows version and build info`,
	Run: func(cmd *cobra.Command, args []string) {

		ui.Logo()
		ui.Info("Version", Version)
		ui.Info("Commit", Commit)
		ui.Info("Built by", BuiltBy)
		ui.Info("Build date", Date)

	},
}
