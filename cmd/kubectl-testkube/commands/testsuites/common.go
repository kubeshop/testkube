package testsuites

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
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

func uiPrintExecutionStatus(execution testkube.TestSuiteExecution) {
	if execution.Status == nil {
		return
	}

	switch true {
	case execution.IsQueued():
		ui.Warn("Test Suite queued for execution")

	case execution.IsRunning():
		ui.Warn("Test Suite execution started")

	case execution.IsPassed():
		ui.Success("Test Suite execution completed with sucess in " + execution.Duration)

	case execution.IsFailed():
		ui.UseStderr()
		ui.Errf("Test Suite execution failed")
		os.Exit(1)
	}

	ui.NL()
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

	err = json.Unmarshal([]byte(*data), &options)
	if err != nil {
		return options, err
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

	variables, err := common.CreateVariables(cmd)
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

	options.Schedule = schedule
	options.ExecutionRequest = &testkube.TestSuiteExecutionRequest{
		Variables:  variables,
		Name:       cmd.Flag("execution-name").Value.String(),
		HttpProxy:  cmd.Flag("http-proxy").Value.String(),
		HttpsProxy: cmd.Flag("https-proxy").Value.String(),
		Timeout:    timeout,
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
		err = json.Unmarshal([]byte(*data), &options)
		if err != nil {
			return options, err
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
		variables, err := common.CreateVariables(cmd)
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
