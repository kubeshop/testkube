package testsuites

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	apiclientv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func printExecution(execution testkube.TestSuiteExecution, startTime time.Time) {
	if execution.TestSuite != nil {
		ui.Warn("Name          :", execution.TestSuite.Name)
	}

	if execution.Id != "" {
		ui.Warn("Execution ID  :", execution.Id)
		ui.Warn("Execution name:", execution.Name)
	}

	if execution.Status != nil {
		ui.Warn("Status        :", string(*execution.Status))
	}

	if execution.Id != "" {
		ui.Warn("Duration:", execution.CalculateDuration().String()+"\n")
		ui.Table(execution, os.Stdout)
	}

	ui.NL()
	ui.NL()
}

func uiPrintExecutionStatus(client apiclientv1.Client, execution testkube.TestSuiteExecution) error {
	if execution.Status == nil {
		return nil
	}

	switch true {
	case execution.IsQueued():
		ui.Warn("Test Suite queued for execution")

	case execution.IsRunning():
		ui.Warn("Test Suite execution started")

	case execution.IsPassed():
		ui.Success("Test Suite execution completed with sucess in " + execution.Duration)

		info, err := client.GetServerInfo()
		ui.ExitOnError("getting server info", err)

		render.PrintTestSuiteExecutionURIs(&execution, info.DashboardUri)

	case execution.IsFailed():
		ui.UseStderr()
		ui.Errf("Test Suite execution failed")

		info, err := client.GetServerInfo()
		ui.ExitOnError("getting server info", err)

		render.PrintTestSuiteExecutionURIs(&execution, info.DashboardUri)
		return errors.New("failed test suite")
	}

	ui.NL()
	return nil
}

func uiShellTestSuiteGetCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube get tse "+id,
	)

	ui.NL()
}

func uiShellTestSuiteWatchCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube watch tse "+id,
	)

	ui.NL()
}

// NewTestSuiteUpsertOptionsFromFlags creates test suite upsert options from command flags
func NewTestSuiteUpsertOptionsFromFlags(cmd *cobra.Command) (options apiclientv1.UpsertTestSuiteOptions, err error) {
	data, err := common.NewDataFromFlags(cmd)
	if err != nil {
		return options, err
	}

	if data == nil {
		return options, fmt.Errorf("empty test suite content")
	}

	if err = json.Unmarshal([]byte(*data), &options); err != nil {
		ui.Debug("json unmarshaling", err.Error())
	}

	emptyBatch := true
	for _, step := range options.Steps {
		if len(step.Execute) != 0 {
			emptyBatch = false
			break
		}
	}

	if emptyBatch {
		var testSuite testkube.TestSuiteUpsertRequestV2
		err = json.Unmarshal([]byte(*data), &testSuite)
		if err != nil {
			return options, err
		}

		options = apiclientv1.UpsertTestSuiteOptions(*testSuite.ToTestSuiteUpsertRequest())
		if len(options.Steps) == 0 {
			return options, fmt.Errorf("no test suite batch steps provided")
		}
	}

	for _, step := range options.Steps {
		if len(step.Execute) == 0 {
			return options, fmt.Errorf("no steps defined for batch step")
		}
	}

	name := cmd.Flag("name").Value.String()
	if name != "" {
		options.Name = name
	}

	labels, err := cmd.Flags().GetStringToString("label")
	if err != nil {
		return options, err
	}

	options.Namespace = cmd.Flag("namespace").Value.String()
	options.Labels = labels

	crdOnly, err := cmd.Flags().GetBool("crd-only")
	if err != nil {
		return options, err
	}

	disableSecretCreation := false
	if !crdOnly {
		client, _, err := common.GetClient(cmd)
		if err != nil {
			return options, err
		}

		info, err := client.GetServerInfo()
		if err != nil {
			return options, err
		}

		disableSecretCreation = info.DisableSecretCreation
	}

	variables, err := common.CreateVariables(cmd, disableSecretCreation)
	if err != nil {
		return options, fmt.Errorf("invalid variables %w", err)
	}

	timeout, err := cmd.Flags().GetInt32("timeout")
	if err != nil {
		return options, err
	}

	schedule := cmd.Flag("schedule").Value.String()
	if err = validateSchedule(schedule); err != nil {
		return options, fmt.Errorf("validating schedule %w", err)
	}

	jobTemplateReference := cmd.Flag("job-template-reference").Value.String()
	cronJobTemplateReference := cmd.Flag("cronjob-template-reference").Value.String()
	scraperTemplateReference := cmd.Flag("scraper-template-reference").Value.String()
	pvcTemplateReference := cmd.Flag("pvc-template-reference").Value.String()

	options.Schedule = schedule
	options.ExecutionRequest = &testkube.TestSuiteExecutionRequest{
		Variables:                variables,
		Name:                     cmd.Flag("execution-name").Value.String(),
		HttpProxy:                cmd.Flag("http-proxy").Value.String(),
		HttpsProxy:               cmd.Flag("https-proxy").Value.String(),
		Timeout:                  timeout,
		JobTemplateReference:     jobTemplateReference,
		CronJobTemplateReference: cronJobTemplateReference,
		ScraperTemplateReference: scraperTemplateReference,
		PvcTemplateReference:     pvcTemplateReference,
	}

	var fields = []struct {
		source      string
		destination *string
	}{
		{
			cmd.Flag("job-template").Value.String(),
			&options.ExecutionRequest.JobTemplate,
		},
		{
			cmd.Flag("cronjob-template").Value.String(),
			&options.ExecutionRequest.CronJobTemplate,
		},
		{
			cmd.Flag("scraper-template").Value.String(),
			&options.ExecutionRequest.ScraperTemplate,
		},
		{
			cmd.Flag("pvc-template").Value.String(),
			&options.ExecutionRequest.PvcTemplate,
		},
	}

	for _, field := range fields {
		if field.source != "" {
			b, err := os.ReadFile(field.source)
			if err != nil {
				return options, err
			}

			*field.destination = string(b)
		}
	}

	return options, nil
}

// NewTestSuiteUpdateOptionsFromFlags creates test suite update options from command flags
func NewTestSuiteUpdateOptionsFromFlags(cmd *cobra.Command) (options apiclientv1.UpdateTestSuiteOptions, err error) {
	data, err := common.NewDataFromFlags(cmd)
	if err != nil {
		return options, err
	}

	if data != nil {
		if err = json.Unmarshal([]byte(*data), &options); err != nil {
			ui.Debug("json unmarshaling", err.Error())
		}

		if options.Steps != nil {
			emptyBatch := true
			for _, step := range *options.Steps {
				if len(step.Execute) != 0 {
					emptyBatch = false
					break
				}
			}

			if emptyBatch {
				var testSuite testkube.TestSuiteUpdateRequestV2
				err = json.Unmarshal([]byte(*data), &testSuite)
				if err != nil {
					return options, err
				}

				options = apiclientv1.UpdateTestSuiteOptions(*testSuite.ToTestSuiteUpdateRequest())
			}

		}
	}

	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"name",
			&options.Name,
		},
		{
			"namespace",
			&options.Namespace,
		},
	}

	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
		}
	}

	if cmd.Flag("schedule").Changed {
		schedule := cmd.Flag("schedule").Value.String()
		if err = validateSchedule(schedule); err != nil {
			return options, fmt.Errorf("validating schedule %w", err)
		}

		options.Schedule = &schedule
	}

	if cmd.Flag("label").Changed {
		labels, err := cmd.Flags().GetStringToString("label")
		if err != nil {
			return options, err
		}

		options.Labels = &labels
	}

	var executionRequest testkube.TestSuiteExecutionUpdateRequest
	var nonEmpty bool
	if cmd.Flag("variable").Changed || cmd.Flag("secret-variable").Changed || cmd.Flag("secret-variable-reference").Changed {
		client, _, err := common.GetClient(cmd)
		if err != nil {
			return options, err
		}

		info, err := client.GetServerInfo()
		if err != nil {
			return options, err
		}

		variables, err := common.CreateVariables(cmd, info.DisableSecretCreation)
		if err != nil {
			return options, fmt.Errorf("invalid variables %w", err)
		}

		executionRequest.Variables = &variables
		nonEmpty = true
	}

	if cmd.Flag("timeout").Changed {
		timeout, err := cmd.Flags().GetInt32("timeout")
		if err != nil {
			return options, err
		}

		executionRequest.Timeout = &timeout
		nonEmpty = true
	}

	var values = []struct {
		source      string
		destination **string
	}{
		{
			"job-template",
			&executionRequest.JobTemplate,
		},
		{
			"cronjob-template",
			&executionRequest.CronJobTemplate,
		},
		{
			"scraper-template",
			&executionRequest.ScraperTemplate,
		},
		{
			"pvc-template",
			&executionRequest.PvcTemplate,
		},
	}

	for _, value := range values {
		if cmd.Flag(value.source).Changed {
			data := ""
			name := cmd.Flag(value.source).Value.String()
			if name != "" {
				b, err := os.ReadFile(name)
				if err != nil {
					return options, err
				}

				data = string(b)
			}

			*value.destination = &data
			nonEmpty = true
		}
	}

	var executionFields = []struct {
		name        string
		destination **string
	}{
		{
			"execution-name",
			&executionRequest.Name,
		},
		{
			"http-proxy",
			&executionRequest.HttpProxy,
		},
		{
			"https-proxy",
			&executionRequest.HttpsProxy,
		},
		{
			"job-template-reference",
			&executionRequest.JobTemplateReference,
		},
		{
			"cronjob-template-reference",
			&executionRequest.CronJobTemplateReference,
		},
		{
			"scraper-template-reference",
			&executionRequest.ScraperTemplateReference,
		},
		{
			"pvc-template-reference",
			&executionRequest.PvcTemplateReference,
		},
	}

	for _, field := range executionFields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
			nonEmpty = true
		}
	}

	if nonEmpty {
		value := (&executionRequest)
		options.ExecutionRequest = &value
	}

	return options, nil
}

func DownloadArtifacts(id, dir, format string, masks []string, client apiclientv1.Client) {
	testSuiteExecution, err := client.GetTestSuiteExecution(id)
	ui.ExitOnError("getting test suite execution ", err)

	for _, execution := range testSuiteExecution.ExecuteStepResults {
		for _, step := range execution.Execute {
			if step.Execution != nil && step.Step != nil && step.Step.Test != "" {
				if step.Execution.IsPassed() || step.Execution.IsFailed() {
					tests.DownloadTestArtifacts(step.Execution.Id, filepath.Join(dir, step.Execution.TestName+"-"+step.Execution.Id), format, masks, client)
				}
			}
		}
	}
}
