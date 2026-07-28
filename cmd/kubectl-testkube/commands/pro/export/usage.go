package export

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export Testkube Enterprise usage data",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(NewUsageCmd())
	return cmd
}

func NewUsageCmd() *cobra.Command {
	var (
		kubeContext     string
		release         string
		valuesFiles     []string
		helmSet         map[string]string
		helmArg         map[string]string
		chartVersion    string
		chartPath       string
		output          string
		timeout         string
		createNamespace bool
		keepRelease     bool
		dryRun          bool
		autoConfig      bool
		noAutoConfig    bool
	)

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Install the usage-export chart, download the zip, and print follow-up steps",
		Long: `Installs the standalone testkube-usage-export Helm chart, waits for the export Job,
copies the resulting zip locally, removes the release, and prints follow-up steps.

When no -f values file is provided, configuration is auto-detected from the Testkube Enterprise
cloud-api deployment in the target namespace (database, credentials, license).`,
		PreRun: func(cmd *cobra.Command, args []string) {
			validator.PersistentPreRunVersionCheck(cmd, common.Version)
		},
		Run: func(cmd *cobra.Command, args []string) {
			ns, err := cmd.Flags().GetString("namespace")
			if err != nil || ns == "" {
				ns, _ = cmd.Root().PersistentFlags().GetString("namespace")
			}

			opts := Options{
				Namespace:       ns,
				KubeContext:     kubeContext,
				Release:         release,
				ValuesFiles:     valuesFiles,
				HelmSet:         helmSet,
				HelmArg:         helmArg,
				ChartVersion:    chartVersion,
				ChartPath:       chartPath,
				Output:          output,
				Timeout:         timeout,
				CreateNamespace: createNamespace,
				KeepRelease:     keepRelease,
				DryRun:          dryRun,
			}

			if opts.Namespace == "" {
				opts.Namespace = "testkube"
			}
			if opts.Release == "" {
				opts.Release = defaultRelease
			}

			if cliErr := applyAutoConfig(&opts, autoConfig, noAutoConfig); cliErr != nil {
				finishUsageExport(opts, "", "", cliErr)
			}

			if !dryRun {
				currentContext, cliErr := common.GetCurrentKubernetesContext()
				if cliErr != nil {
					finishUsageExport(opts, "", "", cliErr)
				}
				ui.Info("Kubernetes context:", currentContext)
				if kubeContext != "" {
					ui.Info("Using kube context override:", kubeContext)
				}
				ui.Info("Namespace:", opts.Namespace)
				ui.NL()
			}

			spinner := ui.NewSpinner("Installing usage export chart...")
			jobName, cliErr := Install(opts)
			if cliErr != nil {
				spinner.Fail()
				finishUsageExport(opts, jobName, "", cliErr)
			}
			spinner.Success("Chart installed, Job:", jobName)

			spinner = ui.NewSpinner("Waiting for usage export and downloading zip...")
			localPath, podName, cliErr := WaitAndDownload(context.Background(), opts, jobName)
			if cliErr != nil {
				spinner.Fail()
				finishUsageExport(opts, jobName, podName, cliErr)
			}
			spinner.Success("Downloaded:", localPath)

			if !keepRelease {
				spinner = ui.NewSpinner("Cleaning up usage export release...")
				if cliErr := Uninstall(opts); cliErr != nil {
					spinner.Fail()
					finishUsageExport(opts, jobName, podName, cliErr)
				}
				spinner.Success("Removed Helm release:", opts.Release)
			}

			if dryRun {
				printRunVersions(opts)
				ui.Info("Dry run complete — no resources were created.")
				os.Exit(0)
			}

			PrintInstructions(opts, localPath)
			printRunVersions(opts)
		},
	}

	cmd.Flags().StringVar(&kubeContext, "context", "", "Kubernetes context")
	cmd.Flags().StringVar(&release, "release", defaultRelease, "Helm release name")
	cmd.Flags().StringArrayVarP(&valuesFiles, "values", "f", nil, "Helm values file(s)")
	cmd.Flags().StringToStringVar(&helmSet, "helm-set", nil, "Helm --set key=value")
	cmd.Flags().StringToStringVar(&helmArg, "helm-arg", nil, "Extra Helm argument in form key=value")
	cmd.Flags().StringVar(&chartVersion, "chart-version", "", "Pin the testkube-usage-export chart version")
	cmd.Flags().StringVar(&chartPath, "chart-path", "", "Local chart path (dev only; skips remote repo)")
	cmd.Flags().StringVar(&output, "output", "", "Local path for the downloaded zip")
	cmd.Flags().StringVar(&timeout, "timeout", "15m", "Maximum time to wait for export completion")
	cmd.Flags().BoolVar(&createNamespace, "create-namespace", true, "Create namespace if it does not exist")
	cmd.Flags().BoolVar(&keepRelease, "keep-release", false, "Keep the Helm release after download (default: uninstall when finished)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print helm/kubectl commands only")
	cmd.Flags().BoolVar(&autoConfig, "auto-config", false, "Discover DB/license config from the enterprise cloud-api deployment (default when no -f is passed)")
	cmd.Flags().BoolVar(&noAutoConfig, "no-auto-config", false, "Disable auto-config; requires -f values.yaml")

	return cmd
}
