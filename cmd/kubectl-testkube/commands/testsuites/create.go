package testsuites

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
)

func NewCreateTestSuitesCmd() *cobra.Command {

	var (
		name      string
		file      string
		labels    map[string]string
		variables map[string]string
		schedule  string
	)

	cmd := &cobra.Command{
		Use:     "testsuite",
		Aliases: []string{"testsuites", "ts"},
		Short:   "Create new TestSuite",
		Long:    `Create new TestSuite Custom Resource`,
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

			if name == "" {
				ui.Failf("pass valid test suite name (in '--name' flag)")
			}

			var options testkube.TestSuiteUpsertRequest

			err = json.Unmarshal(content, &options)
			ui.ExitOnError("Invalid file content", err)

			if name != "" {
				options.Name = name
			}

			client, namespace := common.GetClient(cmd)
			options.Namespace = namespace

			test, _ := client.GetTestSuite(options.Name)
			if options.Name == test.Name {
				ui.Failf("TestSuite with name '%s' already exists in namespace %s", options.Name, options.Namespace)
			}

			options.Labels = labels

			variables, err := common.CreateVariables(cmd)
			ui.ExitOnError("Invalid variables", err)
			options.Variables = variables

			options.Schedule = cmd.Flag("schedule").Value.String()
			err = validateSchedule(options.Schedule)
			ui.ExitOnError("validating schedule", err)

			test, err = client.CreateTestSuite((apiClient.UpsertTestSuiteOptions(options)))
			ui.ExitOnError("creating TestSuite "+options.Name+" in namespace "+options.Namespace, err)
			ui.Success("TestSuite created", options.Name)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON test suite file - will be read from stdin if not specified, look at testkube.TestUpsertRequest")
	cmd.Flags().StringVar(&name, "name", "", "Set/Override test suite name")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringToStringVarP(&variables, "variable", "p", nil, "param key value pair: --variable key1=value1")
	cmd.Flags().StringVarP(&schedule, "schedule", "", "", "test suite schedule in a cronjob form: * * * * *")

	return cmd
}

func validateSchedule(schedule string) error {
	if schedule == "" {
		return nil
	}

	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := specParser.Parse(schedule); err != nil {
		return err
	}

	return nil
}
