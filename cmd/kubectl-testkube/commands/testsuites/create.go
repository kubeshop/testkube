package testsuites

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateTestSuitesCmd() *cobra.Command {

	var (
		name string
		file string
		tags []string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create new test suite",
		Long:    `Create new TestSuite Custom Resource, `,
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

			var options testkube.TestSuiteUpsertRequest

			err = json.Unmarshal(content, &options)
			ui.ExitOnError("Invalid file content", err)

			if name != "" {
				options.Name = name
			}

			client, _ := common.GetClient(cmd)

			test, _ := client.GetTestSuite(options.Name, options.Namespace)
			if options.Name == test.Name {
				ui.Failf("TestSuite with name '%s' already exists in namespace %s", options.Name, options.Namespace)
			}

			options.Tags = tags

			test, err = client.CreateTestSuite((apiClient.UpsertTestSuiteOptions(options)))
			ui.ExitOnError("creating TestSuite "+options.Name+" in namespace "+options.Namespace, err)
			ui.Success("TestSuite created", options.Name)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON test suite file - will be read from stdin if not specified, look at testkube.TestUpsertRequest")
	cmd.Flags().StringVar(&name, "name", "", "Set/Override test suite name")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "comma separated list of tags: --tags tag1,tag2,tag3")

	return cmd
}
