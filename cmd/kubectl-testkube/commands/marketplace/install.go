package marketplace

import (
	"errors"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows"
	"github.com/kubeshop/testkube/pkg/marketplace"
)

// isStdinTTY reports whether stdin is attached to a terminal. It is a
// package-level var so tests can stub it without mocking the OS.
var isStdinTTY = func() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func NewInstallCmd() *cobra.Command {
	var (
		name        string
		update      bool
		dryRun      bool
		interactive bool
		run         bool
		follow      bool
		setFlags    []string
	)

	cmd := &cobra.Command{
		Use:   "install <name>",
		Args:  cobra.ExactArgs(1),
		Short: "Install a marketplace TestWorkflow into the cluster",
		Long: `Downloads a TestWorkflow from the Testkube Marketplace, applies any --set
parameter overrides to its spec.config defaults, and creates (or updates) the
TestWorkflow in the target namespace.

When run on a terminal the command prompts for every parameter the workflow
exposes; values supplied via --set are used as the prompt default, and empty
input keeps the current value. Parameters marked sensitive are read with
masked input and their current value is never echoed. Pass --interactive=false
(or run without a TTY, e.g. in CI) to skip prompting and use the defaults plus
any --set overrides as-is. --interactive=true forces prompting even off a TTY.

After a successful create/update the command asks whether to run the workflow
immediately. Use --run=true to skip the prompt and trigger a run, or
--run=false to skip both the prompt and the run (useful for CI/automation).
Pass -f/--follow to stream logs and wait for completion (implies --run=true
when --run was not supplied).`,

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

			if shouldPromptForParameters(cmd, interactive, len(params) > 0) {
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

			resolvedName := testworkflows.CreateOrUpdateFromBytes(cmd, updated, name, update, dryRun)

			// dryRun already exited inside CreateOrUpdateFromBytes; this guard
			// is defensive in case that ever changes.
			if dryRun {
				return
			}

			shouldRun, decided := resolveRunDecision(cmd, run, follow, ptermPrompter{})
			if !decided || !shouldRun {
				return
			}
			testworkflows.RunWorkflowByName(cmd, resolvedName, follow)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "override the TestWorkflow name")
	cmd.Flags().BoolVar(&update, "update", false, "update, if test workflow already exists")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate the workflow (with overrides applied) without creating it")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "prompt for every spec.config parameter (default: auto — prompt when attached to a terminal). Use --interactive=false to force non-interactive, --interactive to force prompting.")
	cmd.Flags().BoolVar(&run, "run", false, "run the workflow after install (default: ask). Use --run=true to skip the prompt and run, --run=false to skip both prompt and run")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow the execution (stream logs and wait for completion); implies --run=true when --run is not supplied")
	cmd.Flags().StringArrayVar(&setFlags, "set", nil, "override a spec.config parameter, in key=value form (repeatable)")

	return cmd
}

// shouldPromptForParameters decides whether to ask the user for parameter
// values. The --interactive flag is tristate:
//
//   - explicitly set (--interactive / -i / --interactive=false) → flag wins
//   - not set                                                   → prompt only
//     when stdin is a TTY and the workflow exposes at least one parameter
//
// This keeps `testkube marketplace install <name>` self-guiding on a terminal
// while staying quiet in CI pipelines where stdin is redirected.
func shouldPromptForParameters(cmd *cobra.Command, interactiveFlag, hasParams bool) bool {
	if cmd.Flags().Changed("interactive") {
		return interactiveFlag
	}
	if !hasParams {
		return false
	}
	return isStdinTTY()
}

// resolveRunDecision determines whether to run the workflow after install.
//
// Precedence (first match wins):
//  1. --run was supplied explicitly (either true or false) — its value wins.
//  2. -f/--follow was supplied without --run — imply run=true; following an
//     install that did not trigger a run would never yield any output.
//  3. Otherwise ask the user via the prompter.
//
// The second return value is true when a decision was reached (it is false
// only when prompting failed, in which case the caller should skip running).
func resolveRunDecision(cmd *cobra.Command, runFlag, followFlag bool, prompter Prompter) (shouldRun, decided bool) {
	if cmd.Flags().Changed("run") {
		return runFlag, true
	}
	if followFlag {
		return true, true
	}
	answer, err := prompter.Confirm("Run the workflow now?", true)
	if err != nil {
		common.HandleCLIError(common.NewCLIError(
			common.TKErrMarketplaceInvalidParameter,
			"Failed to read run confirmation",
			"Re-run with --run=true or --run=false to skip the prompt.",
			err,
		))
		return false, false
	}
	return answer, true
}
