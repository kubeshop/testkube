package commands

import (
	"github.com/pterm/pterm"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var disconnectOpts = HelmUpgradeOrInstalTestkubeOptions{DryRun: true}

func NewDisconnectCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "disconnect",
		Aliases: []string{"disconnect-cloud"},
		Short:   "Switch back to Testkube OSS mode, based on active .kube/config file",
		Run:     cloudDisconnect,
	}

	PopulateUpgradeInstallFlags(cmd, &disconnectOpts)

	return cmd
}

func cloudDisconnect(cmd *cobra.Command, args []string) {

	h1 := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.Bold)).WithTextStyle(pterm.NewStyle(pterm.FgLightMagenta)).WithMargin(0)
	h2 := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.Bold)).WithTextStyle(pterm.NewStyle(pterm.FgLightGreen)).WithMargin(0)
	// text := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.FgDarkGray)).
	// 	WithTextStyle(pterm.NewStyle(pterm.BgDefault, pterm.FgGray)).WithMargin(0)

	text := pterm.DefaultParagraph.WithMaxWidth(100)

	h1.Println("Disconnecting your cloud environment:")
	text.Println("Rolling back to your clusters testkube OSS installation\nIf you need more details click into following link: " + docsUrl)
	h2.Println("You can safely switch between connecting Cloud and disconnecting without losing your data.")

	cfg, err := config.Load()
	if err != nil {
		pterm.Error.Printfln("Failed to load config file: %s", err.Error())
		return
	}

	var clusterContext string
	if cfg.ContextType == config.ContextTypeKubeconfig {
		clusterContext, err = GetCurrentKubernetesContext()
		if err != nil {
			pterm.Error.Printfln("Failed to get current kubernetes context: %s", err.Error())
			return
		}
	}

	// TODO: implement context info
	h1.Println("Current status of your Testkube instance")
	ui.InfoGrid(map[string]string{
		"Context":   string(cfg.ContextType),
		"Namespace": cfg.Namespace,
		"Cluster":   clusterContext,
	})

	if ok := ui.Confirm("Continue"); !ok {
		return
	}

	// resurrect all scaled down deployments
	disconnectOpts.NoDashboard = false
	disconnectOpts.NoMinio = false
	disconnectOpts.NoMongo = false
	disconnectOpts.MinioReplicas = 1
	disconnectOpts.MongoReplicas = 1
	disconnectOpts.DashboardReplicas = 1

	ui.NL(2)

	spinner := ui.NewSpinner("Connecting back to Testkube OSS")

	err = HelmUpgradeOrInstalTestkube(disconnectOpts)
	ui.ExitOnError("Installing Testkube Cloud", err)
	spinner.Success()

	spinner = ui.NewSpinner("Waking up Testkube OSS components")
	// let's scale down deployment of mongo
	if disconnectOpts.MongoReplicas == 0 {
		KubectlScaleDeployment(disconnectOpts.Namespace, "testkube-mongo", disconnectOpts.MongoReplicas)
		KubectlScaleDeployment(disconnectOpts.Namespace, "testkube-minio-testkube", disconnectOpts.MinioReplicas)
		KubectlScaleDeployment(disconnectOpts.Namespace, "testkube-dashboard", disconnectOpts.DashboardReplicas)
	}
	spinner.Success()
}
