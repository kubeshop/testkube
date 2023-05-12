package cloud

import (
	"strings"

	"github.com/pterm/pterm"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDisconnectCmd() *cobra.Command {

	var opts common.HelmOptions

	cmd := &cobra.Command{
		Use:     "disconnect",
		Aliases: []string{"d"},
		Short:   "Switch back to Testkube OSS mode, based on active .kube/config file",
		Run: func(cmd *cobra.Command, args []string) {

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

			ui.NL(2)

			spinner := ui.NewSpinner("Connecting back to Testkube OSS")

			err = common.HelmUpgradeOrInstalTestkube(opts)
			ui.ExitOnError("Installing Testkube Cloud", err)
			spinner.Success()

			// let's scale down deployment of mongo
			if opts.MongoReplicas > 0 {
				spinner = ui.NewSpinner("Scaling up MongoDB")
				common.KubectlScaleDeployment(opts.Namespace, "testkube-mongodb", opts.MongoReplicas)
				spinner.Success()
			}
			if opts.MinioReplicas > 0 {
				spinner = ui.NewSpinner("Scaling up MinIO")
				common.KubectlScaleDeployment(opts.Namespace, "testkube-minio-testkube", opts.MinioReplicas)
				spinner.Success()
			}
			if opts.DashboardReplicas > 0 {
				spinner = ui.NewSpinner("Scaling up Dashbaord")
				common.KubectlScaleDeployment(opts.Namespace, "testkube-dashboard", opts.DashboardReplicas)
				spinner.Success()
			}

			ui.NL()
			ui.Success("Disconnect finished successfully")
			ui.NL()
			ui.ShellCommand("You can now open your local Dashboard and validate the successfull disconnect:", "testkube dashboard")
		},
	}

	// populate options
	cmd.Flags().StringVar(&opts.Chart, "chart", "kubeshop/testkube", "chart name (usually you don't need to change it)")
	cmd.Flags().StringVar(&opts.Name, "name", "testkube", "installation name (usually you don't need to change it)")
	cmd.Flags().StringVar(&opts.Namespace, "namespace", "testkube", "namespace where to install")
	cmd.Flags().StringVar(&opts.Values, "values", "", "path to Helm values file")

	cmd.Flags().BoolVar(&opts.NoMinio, "no-minio", false, "don't install MinIO")
	cmd.Flags().BoolVar(&opts.NoDashboard, "no-dashboard", false, "don't install dashboard")
	cmd.Flags().BoolVar(&opts.NoMongo, "no-mongo", false, "don't install MongoDB")

	cmd.Flags().IntVar(&opts.MinioReplicas, "minio-replicas", 1, "MinIO replicas")
	cmd.Flags().IntVar(&opts.MongoReplicas, "mongo-replicas", 1, "MongoDB replicas")
	cmd.Flags().IntVar(&opts.DashboardReplicas, "dashboard-replicas", 1, "Dashboard replicas")

	cmd.Flags().BoolVar(&opts.NoConfirm, "no-confirm", false, "don't ask for confirmation - unatended installation mode")

	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "dry run mode - only print commands that would be executed")

	return cmd
}
