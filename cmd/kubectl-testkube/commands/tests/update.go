package tests

import (
	"encoding/json"
	"io/ioutil"
	"os"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpdateTestsCmd() *cobra.Command {

	var (
		file string
		tags []string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Test",
		Long:  `Update Test Custom Resource Definitions, `,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			var content []byte
			var err error

			if file != "" {
				// read test content
				content, err = ioutil.ReadFile(file)
				ui.ExitOnError("reading file"+file, err)
			} else if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
				content, err = ioutil.ReadAll(os.Stdin)
				ui.ExitOnError("reading stdin", err)
			}

			var options apiClient.UpsertTestOptions

			json.Unmarshal(content, &options)

			client, _ := GetClient(cmd)

			test, _ := client.GetTest(options.Name, options.Namespace)
			if options.Name == test.Name {
				ui.Failf("Test with name '%s' already exists in namespace %s", options.Name, options.Namespace)
			}

			// if tags are not passed don't overwrite existing tags
			// TODO: figure out how to remove tags from test
			if tags != nil {
				options.Tags = tags
			}

			test, err = client.UpdateTest(options)
			ui.ExitOnError("updating test "+options.Name+" in namespace "+options.Namespace, err)
			ui.Success("Test created", options.Name)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON test file - will be read from stdin if not specified, look at testkube.TestUpsertRequest")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "--tags 1,2,3")

	return cmd
}
