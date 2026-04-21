package marketplace

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/marketplace"
	"github.com/kubeshop/testkube/pkg/ui"
)

// categorySummary is the per-category aggregate rendered by `marketplace
// categories`. Exported field names keep encoding/json and yaml.v3 output
// consistent with the rest of the marketplace commands.
type categorySummary struct {
	Category   string   `json:"category" yaml:"category"`
	Count      int      `json:"count" yaml:"count"`
	Components []string `json:"components" yaml:"components"`
}

func NewCategoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "categories",
		Aliases: []string{"cats"},
		Args:    cobra.NoArgs,
		Short:   "List marketplace categories with workflow counts and components",
		Long: `Aggregates the marketplace catalog into one row per category, showing how many
workflows the category contains and the distinct components inside it.`,

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

			summaries := aggregateCategories(workflows)

			outputType := cmd.Flag("output").Value.String()
			if outputType == "json" || outputType == "yaml" || outputType == "go" {
				err = render.List(cmd, summaries, os.Stdout)
				ui.PrintOnError("Rendering list", err)
				return
			}

			err = render.List(cmd, categoriesTable(summaries), os.Stdout)
			ui.PrintOnError("Rendering list", err)
		},
	}

	return cmd
}

// aggregateCategories collapses workflows into one row per category. Component
// lists are deduped and sorted; categories are sorted alphabetically. The
// returned slice is always non-nil so callers can encode it as `[]` rather
// than `null` when the catalog is empty.
func aggregateCategories(workflows []marketplace.Workflow) []categorySummary {
	type bucket struct {
		count      int
		components map[string]struct{}
	}
	buckets := make(map[string]*bucket)
	for _, w := range workflows {
		b, ok := buckets[w.Category]
		if !ok {
			b = &bucket{components: make(map[string]struct{})}
			buckets[w.Category] = b
		}
		b.count++
		if w.Component != "" {
			b.components[w.Component] = struct{}{}
		}
	}

	summaries := make([]categorySummary, 0, len(buckets))
	for cat, b := range buckets {
		comps := make([]string, 0, len(b.components))
		for c := range b.components {
			comps = append(comps, c)
		}
		sort.Strings(comps)
		summaries = append(summaries, categorySummary{
			Category:   cat,
			Count:      b.count,
			Components: comps,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Category < summaries[j].Category
	})
	return summaries
}

type categoriesTable []categorySummary

func (t categoriesTable) Table() ([]string, [][]string) {
	header := []string{"CATEGORY", "WORKFLOWS", "COMPONENTS"}
	data := make([][]string, 0, len(t))
	for _, s := range t {
		data = append(data, []string{
			s.Category,
			strconv.Itoa(s.Count),
			strings.Join(s.Components, ", "),
		})
	}
	return header, data
}
