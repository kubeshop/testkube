package marketplace

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/marketplace"
	"github.com/kubeshop/testkube/pkg/ui"
)

// workflowDetail is the structured payload rendered for `marketplace get` so
// that JSON/YAML/go-template consumers get consistent, machine-readable output.
type workflowDetail struct {
	Workflow   marketplace.Workflow    `json:"workflow" yaml:"workflow"`
	Parameters []marketplace.Parameter `json:"parameters" yaml:"parameters"`
	YAML       string                  `json:"yaml,omitempty" yaml:"yaml,omitempty"`
	Readme     string                  `json:"readme,omitempty" yaml:"readme,omitempty"`
}

func NewGetCmd() *cobra.Command {
	var (
		showYAML   bool
		showReadme bool
	)

	cmd := &cobra.Command{
		Use:     "get <name>",
		Aliases: []string{"describe"},
		Args:    cobra.ExactArgs(1),
		Short:   "Show detailed information about a marketplace TestWorkflow",
		Long: `Displays the catalog entry, configurable parameters, and optionally the raw YAML
and readme content for a marketplace TestWorkflow.`,

		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			client := NewClient()

			wf, err := client.GetWorkflow(cmd.Context(), name)
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

			var readme []byte
			if showReadme {
				readme, err = client.GetReadme(cmd.Context(), *wf)
				if err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrMarketplaceFetchFailed,
						"Failed to download readme",
						"",
						err,
					))
					return
				}
			}

			outputType := cmd.Flag("output").Value.String()
			if outputType == "json" || outputType == "yaml" || outputType == "go" {
				detail := workflowDetail{
					Workflow:   *wf,
					Parameters: params,
					Readme:     string(readme),
				}
				if showYAML {
					detail.YAML = string(yamlBytes)
				}
				err = render.Obj(cmd, detail, os.Stdout)
				ui.PrintOnError("Rendering detail", err)
				return
			}

			renderPretty(*wf, params, yamlBytes, readme, showYAML, showReadme)
		},
	}

	cmd.Flags().BoolVar(&showYAML, "show-yaml", false, "include the raw workflow YAML in the output")
	cmd.Flags().BoolVar(&showReadme, "show-readme", false, "include the workflow readme in the output")

	return cmd
}

func renderPretty(wf marketplace.Workflow, params []marketplace.Parameter, yamlBytes, readme []byte, showYAML, showReadme bool) {
	fmt.Printf("Name:         %s\n", wf.Name)
	fmt.Printf("Display Name: %s\n", wf.DisplayName)
	fmt.Printf("Category:     %s\n", wf.Category)
	fmt.Printf("Component:    %s\n", wf.Component)
	if wf.ValidationType != "" {
		fmt.Printf("Validation:   %s\n", wf.ValidationType)
	}
	fmt.Printf("Description:  %s\n", wf.Description)
	if wf.Icon != "" {
		fmt.Printf("Icon:         %s\n", wf.Icon)
	}
	fmt.Println()

	if len(params) == 0 {
		fmt.Println("Parameters:   (none)")
	} else {
		fmt.Println("Parameters:")
		paramsTable := paramsPrettyTable(params)
		ui.NewUI(ui.Verbose, os.Stdout).Table(paramsTable, os.Stdout)
	}

	if showYAML {
		fmt.Println()
		fmt.Println("YAML:")
		fmt.Println(string(yamlBytes))
	}
	if showReadme && len(readme) > 0 {
		fmt.Println()
		fmt.Println("Readme:")
		fmt.Println(string(readme))
	}
}

type paramsPrettyTable []marketplace.Parameter

func (t paramsPrettyTable) Table() ([]string, [][]string) {
	header := []string{"NAME", "TYPE", "DEFAULT", "SENSITIVE", "DESCRIPTION"}
	data := make([][]string, 0, len(t))
	for _, p := range t {
		sensitive := ""
		if p.Sensitive {
			sensitive = "yes"
		}
		data = append(data, []string{p.Key, p.Type, p.Default, sensitive, p.Description})
	}
	return header, data
}
