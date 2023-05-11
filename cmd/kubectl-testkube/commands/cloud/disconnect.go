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
	}

	ui.Properties(summary)

	if ok := ui.Confirm("Shall we disconnect your cloud environment now?"); !ok {
		return
	}

	//         Disconnect your cloud environment:        You can learn more about disconnecting your Testkube instance to the Cloud here:         https://docs.testkube.io/etc...
	// 	You can safely switch between connecting Cloud and disconnecting without losing your data.
	// 	STATUS  Current status of your Testkube instance
	// 	Context:   Cloud
	// 	Cluster:   Cluster name        Namespace: Testkube        Org. name: My-Org-1        Env. name: my-env-1
	// LOGIN   Login        Please open the following link in your browser and log in:         https://cloud.testkube.io/login?redirect_uri=....
	// SANITY  Summary of your setup after disconnecting:         Context:   On premise
	// 	Cluster:   Cluster name        Namespace: Testkube        Minio:     started and scaled up         MongoDB:   started and scaled up
	// 	Dashboard: started and scaled up         Shall we disconnect your cloud environment now?        ● Yes        ○ No        DISCONNECT        ✅ Updating context to local OSS instance         ✅ Starting Minio        ✅ Starting Dashboard UI        ⏳ Starting MongoDB
	// 	✅ Disconnect finished successfully        🎉 Happy testing!
	// 	You can now open your local Dashboard and validate the successfull disconnect:        https://localhost:3823?api_uri=xxxxx
	//         $

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
}
