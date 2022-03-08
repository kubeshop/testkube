package testsuites

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	testkubeapiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpdateTestSuitesCmd() *cobra.Command {

	var (
		file   string
		labels map[string]string
	)

	cmd := &cobra.Command{
		Use:   "testsuite",
		Short: "Update Test Suite",
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

			var options testkubeapiv1.UpsertTestSuiteOptions

			json.Unmarshal(content, &options)

			client, _ := common.GetClient(cmd)

			test, _ := client.GetTestSuite(options.Name, options.Namespace)
			if options.Name == test.Name {
				ui.Failf("Test with name '%s' already exists in namespace %s", options.Name, options.Namespace)
			}

			// if labels are passed and are different from the existing overwrite
			if len(labels) > 0 && !reflect.DeepEqual(test.Labels, labels) {
				options.Labels = labels
			} else {
				options.Labels = test.Labels
			}

			// if labels are not passed don't overwrite existing labels
			// TODO: figure out how to remove labels from test
			if labels != nil {
				options.Labels = labels
			}

			test, err = client.UpdateTestSuite(options)
			ui.ExitOnError("updating test "+options.Name+" in namespace "+options.Namespace, err)
			ui.Success("Test created", options.Name)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON test file - will be read from stdin if not specified, look at testkube.TestUpsertRequest")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
