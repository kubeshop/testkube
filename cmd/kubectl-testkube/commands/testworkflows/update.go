package testworkflows

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewUpdateTestWorkflowExecutionCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:     "testworkflowexecution <executionID>",
		Aliases: []string{"twe", "testworkflows-execution", "testworkflow-execution"},
		Short:   "Update test workflow execution tags",
		Long:    "Update test workflow execution tags. Use --tag key=value to set tags. Replaces all existing tags. Omit --tag to clear all tags.",
		Args:    validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			parsedTags := make(map[string]string)
			for _, t := range tags {
				parts := strings.SplitN(t, "=", 2)
				if len(parts) != 2 {
					ui.Failf("invalid tag format '%s', expected key=value", t)
				}
				parsedTags[parts[0]] = parts[1]
			}

			err = client.UpdateTestWorkflowExecutionTags(executionID, parsedTags)
			ui.ExitOnError(fmt.Sprintf("updating tags for test workflow execution %s", executionID), err)

			ui.SuccessAndExit("Successfully updated test workflow execution tags", executionID)
		},
	}

	cmd.Flags().StringArrayVar(&tags, "tag", []string{}, "tag in key=value format (can be specified multiple times, omit to clear all tags)")

	return cmd
}
