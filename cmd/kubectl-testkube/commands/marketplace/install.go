package marketplace

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows"
	"github.com/kubeshop/testkube/pkg/marketplace"
)

func NewInstallCmd() *cobra.Command {
	var (
		name     string
		update   bool
		dryRun   bool
		setFlags []string
	)

	cmd := &cobra.Command{
		Use:   "install <name>",
		Args:  cobra.ExactArgs(1),
		Short: "Install a marketplace TestWorkflow into the cluster",
		Long: `Downloads a TestWorkflow from the Testkube Marketplace, applies any --set
parameter overrides to its spec.config defaults, and creates (or updates) the
TestWorkflow in the target namespace.`,

		Run: func(cmd *cobra.Command, args []string) {
			workflowName := args[0]
			client := NewClient()

			wf, err := client.GetWorkflow(cmd.Context(), workflowName)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrMarketplaceWorkflowNotFound,
						"Workflow not found",
						"",
						err,
					))
				}
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceFetchFailed,
					"Failed to fetch marketplace catalog",
					"",
					err,
				))
			}

			yamlBytes, err := client.GetWorkflowYAML(cmd.Context(), *wf)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceFetchFailed,
					"Failed to download workflow YAML",
					"",
					err,
				))
			}

			params, err := marketplace.ExtractParameters(yamlBytes)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceInvalidParameter,
					"Failed to parse workflow parameters",
					"",
					err,
				))
			}

			params, err = marketplace.ParseSetFlags(params, setFlags)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceInvalidParameter,
					"Invalid --set value",
					"",
					err,
				))
			}

			updated, err := marketplace.ApplyParameters(yamlBytes, params)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceInvalidParameter,
					"Failed to apply parameters",
					"",
					err,
				))
			}

			testworkflows.CreateOrUpdateFromBytes(cmd, updated, name, update, dryRun)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "override the TestWorkflow name")
	cmd.Flags().BoolVar(&update, "update", false, "update, if test workflow already exists")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate the workflow (with overrides applied) without creating it")
	cmd.Flags().StringArrayVar(&setFlags, "set", nil, "override a spec.config parameter, in key=value form (repeatable)")

	return cmd
}
