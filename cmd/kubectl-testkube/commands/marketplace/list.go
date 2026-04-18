package marketplace

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/marketplace"
	"github.com/kubeshop/testkube/pkg/ui"
)

// descriptionMaxWidth caps the description column for pretty output so long
// strings don't destroy the table layout.
const descriptionMaxWidth = 60

func NewListCmd() *cobra.Command {
	var (
		category  string
		component string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List TestWorkflows available in the marketplace",
		Long:    "Fetches the marketplace catalog and lists available TestWorkflows. Use --category and --component to filter, or --output to switch formats.",

		Run: func(cmd *cobra.Command, _ []string) {
			client := NewClient()
			workflows, err := client.ListWorkflows(cmd.Context())
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrMarketplaceFetchFailed,
					"Failed to fetch marketplace catalog",
					"",
					err,
				))
			}

			filtered := filter(workflows, category, component)

			outputType := cmd.Flag("output").Value.String()
			if outputType == "json" || outputType == "yaml" || outputType == "go" {
				err = render.List(cmd, filtered, os.Stdout)
				ui.PrintOnError("Rendering list", err)
				return
			}

			err = render.List(cmd, workflowsTable(filtered), os.Stdout)
			ui.PrintOnError("Rendering list", err)
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "filter by category (e.g. databases, networking)")
	cmd.Flags().StringVar(&component, "component", "", "filter by component (e.g. redis, postgresql)")

	return cmd
}

func filter(workflows []marketplace.Workflow, category, component string) []marketplace.Workflow {
	if category == "" && component == "" {
		return workflows
	}
	out := make([]marketplace.Workflow, 0, len(workflows))
	for _, w := range workflows {
		if category != "" && !strings.EqualFold(w.Category, category) {
			continue
		}
		if component != "" && !strings.EqualFold(w.Component, component) {
			continue
		}
		out = append(out, w)
	}
	return out
}

type workflowsTable []marketplace.Workflow

func (t workflowsTable) Table() ([]string, [][]string) {
	header := []string{"NAME", "CATEGORY", "COMPONENT", "DESCRIPTION"}
	data := make([][]string, 0, len(t))
	for _, w := range t {
		data = append(data, []string{w.Name, w.Category, w.Component, truncate(w.Description, descriptionMaxWidth)})
	}
	return header, data
}

// truncate caps a string at max characters, counting by runes so multi-byte
// UTF-8 sequences are never split mid-character.
func truncate(s string, max int) string {
	runes := []rune(s)
	if max <= 0 || len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
