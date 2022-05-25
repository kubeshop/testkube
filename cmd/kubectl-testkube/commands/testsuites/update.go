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
		file     string
		name     string
		labels   map[string]string
		schedule string
	)

	cmd := &cobra.Command{
		Use:   "testsuite",
		Short: "Update Test Suite",
		Long:  `Update Test Custom Resource Definitions, `,
		Run: func(cmd *cobra.Command, args []string) {

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

			err = json.Unmarshal(content, &options)
			ui.ExitOnError("Invalid file content", err)

			if name != "" {
				options.Name = name
			}

			client, namespace := common.GetClient(cmd)
			options.Namespace = namespace

			testSuite, _ := client.GetTestSuite(options.Name)
			if options.Name != testSuite.Name {
				ui.Failf("TestSuite with name '%s' not exists in namespace %s", options.Name, options.Namespace)
			}

			// if labels are passed and are different from the existing overwrite
			if len(labels) > 0 && !reflect.DeepEqual(testSuite.Labels, labels) {
				options.Labels = labels
			} else {
				options.Labels = testSuite.Labels
			}

			options.Schedule = cmd.Flag("schedule").Value.String()

			err = validateSchedule(options.Schedule)
			ui.ExitOnError("validating schedule", err)

			testSuite, err = client.UpdateTestSuite(options)
			ui.ExitOnError("updating TestSuite "+options.Name+" in namespace "+options.Namespace, err)
			ui.Success("TestSuite updated", options.Name)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON test file - will be read from stdin if not specified, look at testkube.TestUpsertRequest")
	cmd.Flags().StringVar(&name, "name", "", "Set/Override test suite name")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&schedule, "schedule", "", "", "test suite schedule in a cronjob form: * * * * *")

	return cmd
}
