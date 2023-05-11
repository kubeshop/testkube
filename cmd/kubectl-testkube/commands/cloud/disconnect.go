package cloud

import (
	"strings"

	"github.com/pterm/pterm"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var disconnectOpts = common.HelmUpgradeOrInstalTestkubeOptions{DryRun: true}

func NewDisconnectCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "disconnect",
		Aliases: []string{"disconnect-cloud"},
		Short:   "Switch back to Testkube OSS mode, based on active .kube/config file",
		Run:     cloudDisconnect,
	}

	common.PopulateUpgradeInstallFlags(cmd, &disconnectOpts)

	return cmd
}

func cloudDisconnect(cmd *cobra.Command, args []string) {
	ui.H1("Disconnecting your cloud environment:")
	ui.Paragraph("Rolling back to your clusters testkube OSS installation")
	ui.Paragraph("If you need more details click into following link: " + docsUrl)
	ui.H2("You can safely switch between connecting Cloud and disconnecting without losing your data.")

	cfg, err := config.Load()
	if err != nil {
		pterm.Error.Printfln("Failed to load config file: %s", err.Error())
		return
	}

	client, _ := common.GetClient(cmd)
	info, err := client.GetServerInfo()
	firstInstall := err != nil && strings.Contains(err.Error(), "not found")
	if err != nil && !firstInstall {
		ui.Failf("Can't get testkube cluster information: %s", err.Error())
	}
	var apiContext string
	if actx, ok := contextDescription[info.Context]; ok {
		apiContext = actx
	}
	var clusterContext string
	if cfg.ContextType == config.ContextTypeKubeconfig {
		clusterContext, err = common.GetCurrentKubernetesContext()
		if err != nil {
			pterm.Error.Printfln("Failed to get current kubernetes context: %s", err.Error())
			return
		}
	}

	// TODO: implement context info
	ui.H1("Current status of your Testkube instance")

	summary := [][]string{
		{"Testkube mode"},
		{"Context", apiContext},
		{"Kubectl context", clusterContext},
		{"Namespace", cfg.Namespace},
		{ui.Separator, ""},

		{"Testkube is connected to cloud organizations environment"},
		{"Organization Id", info.OrgId},
		{"Environment Id", info.EnvId},
	}

	ui.Properties(summary)

	if ok := ui.Confirm("Shall we disconnect your cloud environment now?"); !ok {
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

	err = common.HelmUpgradeOrInstalTestkube(disconnectOpts)
	ui.ExitOnError("Installing Testkube Cloud", err)
	spinner.Success()

	// let's scale down deployment of mongo
	if connectOpts.MongoReplicas > 0 {
		spinner = ui.NewSpinner("Scaling down MongoDB")
		common.KubectlScaleDeployment(connectOpts.Namespace, "testkube-mongodb", connectOpts.MongoReplicas)
		spinner.Success()
	}
	if connectOpts.MinioReplicas > 0 {
		spinner = ui.NewSpinner("Scaling down MinIO")
		common.KubectlScaleDeployment(connectOpts.Namespace, "testkube-minio-testkube", connectOpts.MinioReplicas)
		spinner.Success()
	}
	if connectOpts.DashboardReplicas > 0 {
		spinner = ui.NewSpinner("Scaling down Dashbaord")
		common.KubectlScaleDeployment(connectOpts.Namespace, "testkube-dashboard", connectOpts.DashboardReplicas)
		spinner.Success()
	}

	ui.NL()
	ui.Success("Disconnect finished successfully")
	ui.NL()
	ui.ShellCommand("You can now open your local Dashboard and validate the successfull disconnect:", "testkube dashboard")
}
