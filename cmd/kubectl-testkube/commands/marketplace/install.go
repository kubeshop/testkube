package marketplace

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows"
	"github.com/kubeshop/testkube/pkg/marketplace"
)

func NewInstallCmd() *cobra.Command {
	var (
		name        string
		update      bool
		dryRun      bool
		interactive bool
		setFlags    []string
	)

	cmd := &cobra.Command{
		Use:   "install <name>",
		Args:  cobra.ExactArgs(1),
		Short: "Install a marketplace TestWorkflow into the cluster",
		Long: `Downloads a TestWorkflow from the Testkube Marketplace, applies any --set
parameter overrides to its spec.config defaults, and creates (or updates) the
TestWorkflow in the target namespace.

Use --interactive/-i to be prompted for every parameter the workflow exposes.
Values supplied via --set are used as the prompt default, and empty input
keeps the current value. Parameters marked sensitive are read with masked
input and their current value is never echoed.`,

		Run: func(cmd *cobra.Command, args []string) {
			workflowName := args[0]
			client := NewClient()

			wf, err := client.GetWorkflow(cmd.Context(), workflowName)
			if err != nil {
				if errors.Is(err, marketplace.ErrWorkflowNotFound) {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrMarketplaceWorkflowNotFound,
						"Workflow not found",
						"",
						err,
					))
					return
				}
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceFetchFailed,
					"Failed to fetch marketplace catalog",
					"",
					err,
				))
				return
			}

			yamlBytes, err := client.GetWorkflowYAML(cmd.Context(), *wf)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceFetchFailed,
					"Failed to download workflow YAML",
					"",
					err,
				))
				return
			}

			params, err := marketplace.ExtractParameters(yamlBytes)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceInvalidParameter,
					"Failed to parse workflow parameters",
					"",
					err,
				))
				return
			}

			params, err = marketplace.ParseSetFlags(params, setFlags)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceInvalidParameter,
					"Invalid --set value",
					"",
					err,
				))
				return
			}

			if interactive {
				params, err = promptForParameters(os.Stdout, params, ptermPrompter{})
				if err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrMarketplaceInvalidParameter,
						"Failed to read interactive input",
						"",
						err,
					))
					return
				}
			}

			updated, err := marketplace.ApplyParameters(yamlBytes, params)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceInvalidParameter,
					"Failed to apply parameters",
					"",
					err,
				))
				return
			}

			testworkflows.CreateOrUpdateFromBytes(cmd, updated, name, update, dryRun)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "override the TestWorkflow name")
	cmd.Flags().BoolVar(&update, "update", false, "update, if test workflow already exists")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate the workflow (with overrides applied) without creating it")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "prompt for every spec.config parameter the workflow exposes")
	cmd.Flags().StringArrayVar(&setFlags, "set", nil, "override a spec.config parameter, in key=value form (repeatable)")

	return cmd
}
