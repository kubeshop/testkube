package testworkflows

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows/renderer"
	testkubecfg "github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	common2 "github.com/kubeshop/testkube/internal/common"
	apiclientv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	tclcmd "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/cmd"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	LogTimestampLength = 30 // time.RFC3339Nano without 00:00 timezone
	apiErrorMessage    = "processing error:"
	logsCheckDelay     = 100 * time.Millisecond

	logsRetryAttempts = 10
	logsRetryDelay    = time.Second
)

var (
	NL = []byte("\n")
)

func NewRunTestWorkflowCmd() *cobra.Command {
	var (
		executionName            string
		config                   map[string]string
		watchEnabled             bool
		disableWebhooks          bool
		downloadArtifactsEnabled bool
		downloadDir              string
		format                   string
		masks                    []string
		tags                     map[string]string
		selectors                []string
		serviceName              string
		parallelStepName         string
		serviceIndex             int
		parallelStepIndex        int
		targetMatch              []string
		targetNot                []string
		targetReplicate          []string
	)

	cmd := &cobra.Command{
		Use:     "testworkflow [name]",
		Aliases: []string{"testworkflows", "tw"},
		Short:   "Starts test workflow execution",

		Run: func(cmd *cobra.Command, args []string) {
			outputFlag := cmd.Flag("output")
			outputType := render.OutputPretty
			if outputFlag != nil {
				outputType = render.OutputType(outputFlag.Value.String())
			}

			outputPretty := outputType == render.OutputPretty
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			runContext := telemetry.GetCliRunContext()
			interfaceType := testkube.CICD_TestWorkflowRunningContextInterfaceType
			if runContext == "others|local" {
				runContext = ""
				interfaceType = testkube.CLI_TestWorkflowRunningContextInterfaceType
			}

			cfg, err := testkubecfg.Load()
			ui.ExitOnError("loading config file", err)
			ui.NL()

			var runningContext *testkube.TestWorkflowRunningContext
			// Pro edition only (tcl protected code)
			if cfg.ContextType == testkubecfg.ContextTypeCloud {
				runningContext = tclcmd.GetRunningContext(runContext, cfg.CloudContext.ApiKey, interfaceType)
			}

			request := testkube.TestWorkflowExecutionRequest{
				Name:            executionName,
				Config:          config,
				DisableWebhooks: disableWebhooks,
				Tags:            tags,
				RunningContext:  runningContext,
				Target:          &testkube.ExecutionTarget{},
			}

			if len(targetMatch) > 0 {
				request.Target.Match = make(map[string][]string)
				for _, match := range targetMatch {
					key, values, _ := strings.Cut(match, "=")
					request.Target.Match[key] = common2.MapSlice(strings.Split(values, ","), strings.TrimSpace)
				}
			}
			if len(targetNot) > 0 {
				request.Target.Not = make(map[string][]string)
				for _, match := range targetNot {
					key, values, _ := strings.Cut(match, "=")
					request.Target.Not[key] = common2.MapSlice(strings.Split(values, ","), strings.TrimSpace)
				}
			}
			if len(targetReplicate) > 0 {
				request.Target.Replicate = common2.MapSlice(strings.Split(strings.Join(targetReplicate, ","), ","), strings.TrimSpace)
			}

			var executions []testkube.TestWorkflowExecution
			switch {
			case len(args) > 0:
				name := args[0]

				var execution testkube.TestWorkflowExecution
				execution, err = client.ExecuteTestWorkflow(name, request)
				executions = append(executions, execution)
			case len(selectors) != 0:
				selector := strings.Join(selectors, ",")
				executions, err = client.ExecuteTestWorkflows(selector, request)
			default:
				ui.Failf("Pass Test workflow name or labels to run by labels ")
			}

			if err != nil {
				// User friendly Open Source operation error
				errMessage := err.Error()
				if strings.Contains(errMessage, constants.OpenSourceOperationErrorMessage) {
					startp := strings.LastIndex(errMessage, apiErrorMessage)
					endp := strings.Index(errMessage, constants.OpenSourceOperationErrorMessage)
					if startp != -1 && endp != -1 {
						startp += len(apiErrorMessage)
						operation := ""
						if endp > startp {
							operation = strings.TrimSpace(errMessage[startp:endp])
						}

						err = errors.New(operation + " " + constants.OpenSourceOperationErrorMessage)
					}
				}
			}

			if len(args) > 0 {
				ui.ExitOnError("execute test workflow "+args[0]+" from namespace "+namespace, err)
			} else {
				ui.ExitOnError("execute test workflows "+strings.Join(selectors, ",")+" from namespace "+namespace, err)
			}

			go func() {
				<-cmd.Context().Done()
				if errors.Is(cmd.Context().Err(), context.Canceled) {
					os.Exit(0)
				}
			}()

			for _, execution := range executions {
				err = renderer.PrintTestWorkflowExecution(cmd, os.Stdout, execution)
				ui.ExitOnError("render test workflow execution", err)

				var exitCode = 0
				if outputPretty {
					ui.NL()
					if !execution.FailedToInitialize() {
						if watchEnabled && len(args) > 0 {
							var pServiceName, pParallelStepName *string
							if cmd.Flag("service-name").Changed || cmd.Flag("service-index").Changed {
								pServiceName = &serviceName
							}
							if cmd.Flag("parallel-step-name").Changed || cmd.Flag("parallel-step-index").Changed {
								pParallelStepName = &parallelStepName
							}

							exitCode = uiWatch(execution, pServiceName, serviceIndex, pParallelStepName, parallelStepIndex, client)
							ui.NL()
							if downloadArtifactsEnabled {
								tests.DownloadTestWorkflowArtifacts(execution.Id, downloadDir, format, masks, client, outputPretty)
							}
						} else {
							uiShellWatchExecution(execution.Id)
						}
					}

					execution, err = client.GetTestWorkflowExecution(execution.Id)
					ui.ExitOnError("get execution failed", err)

					render.PrintTestWorkflowExecutionURIs(&execution)
					uiShellGetExecution(execution.Id)
				}

				if exitCode != 0 {
					os.Exit(exitCode)
				}
			}
		},
	}

	cmd.Flags().StringVarP(&executionName, "name", "n", "", "execution name, if empty will be autogenerated")
	cmd.Flags().StringToStringVarP(&config, "config", "", map[string]string{}, "configuration variables in a form of name1=val1 passed to executor")
	cmd.Flags().BoolVarP(&watchEnabled, "watch", "f", false, "watch for changes after start")
	cmd.Flags().BoolVar(&disableWebhooks, "disable-webhooks", false, "disable webhooks for this execution")
	cmd.Flags().MarkDeprecated("enable-webhooks", "enable-webhooks is deprecated")
	cmd.Flags().StringVar(&downloadDir, "download-dir", "artifacts", "download dir")
	cmd.Flags().BoolVarP(&downloadArtifactsEnabled, "download-artifacts", "d", false, "download artifacts automatically")
	cmd.Flags().StringVar(&format, "format", "folder", "data format for storing files, one of folder|archive")
	cmd.Flags().StringArrayVarP(&masks, "mask", "", []string{}, "regexp to filter downloaded files, single or comma separated, like report/.* or .*\\.json,.*\\.js$")
	cmd.Flags().StringToStringVarP(&tags, "tag", "", map[string]string{}, "execution tag adds a tag to execution in form of name1=val1 passed to executor")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label is used to select test workflows to run using key value pair: --label key1=value1 or label expression")
	cmd.Flags().StringVar(&serviceName, "service-name", "", "test workflow service name")
	cmd.Flags().IntVar(&serviceIndex, "service-index", 0, "test workflow service index starting from 0")
	cmd.Flags().StringVar(&parallelStepName, "parallel-step-name", "", "test workflow parallel step name or reference")
	cmd.Flags().IntVar(&parallelStepIndex, "parallel-step-index", 0, "test workflow parallel step index starting from 0")
	cmd.Flags().StringArrayVar(&targetMatch, "target", nil, "runner labels to match")
	cmd.Flags().StringArrayVar(&targetNot, "target-not", nil, "runner labels to not match")
	cmd.Flags().StringArrayVar(&targetReplicate, "target-replicate", nil, "runner labels to replicate over")

	return cmd
}

func getIterationDelay(iteration int) time.Duration {
	if iteration < 5 {
		return 500 * time.Millisecond
	} else if iteration < 100 {
		return 1 * time.Second
	}
	return 5 * time.Second
}

func uiWatch(execution testkube.TestWorkflowExecution, serviceName *string, serviceIndex int,
	parallelStepName *string, parallelStepIndex int, client apiclientv1.Client) int {
	// Wait until the execution will be assigned to some runner
	iteration := 0
	for !execution.Assigned() {
		var err error
		iteration++
		time.Sleep(getIterationDelay(iteration))
		execution, err = client.GetTestWorkflowExecution(execution.Id)
		if err != nil {
			ui.Failf("get execution failed: %v", err)
		}
	}

	// Print final logs in case execution is already finished
	if execution.Result.IsFinished() {
		ui.Info("Getting logs for test workflow execution", execution.Id)

		logs, err := client.GetTestWorkflowExecutionLogs(execution.Id)
		ui.ExitOnError("getting logs from executor", err)

		sigs := flattenSignatures(execution.Signature)

		printRawLogLines(logs, sigs, execution)
		return 0
	}

	var result *testkube.TestWorkflowResult
	var err error

	switch {
	case serviceName != nil:
		found := false
		if execution.Workflow != nil {
			found = execution.Workflow.HasService(*serviceName)
		}

		if !found {
			ui.Failf("unknown service '%s' for test workflow execution %s", *serviceName, execution.Id)
		}

		result, err = watchTestWorkflowServiceLogs(execution.Id, *serviceName, serviceIndex, execution.Signature, client)
	case parallelStepName != nil:
		ref := execution.GetParallelStepReference(*parallelStepName)
		if ref == "" {
			ui.Failf("unknown parallel step '%s' for test workflow execution %s", *parallelStepName, execution.Id)
		}

		result, err = watchTestWorkflowParallelStepLogs(execution.Id, ref, parallelStepIndex, execution.Signature, client)
	default:
		result, err = watchTestWorkflowLogs(execution.Id, execution.Signature, client)
	}

	if result == nil && err == nil {
		err = errors.New("no result found")
	}

	ui.ExitOnError("reading test workflow execution logs", err)

	// Apply the result in the execution
	execution.Result = result
	if result.IsFinished() {
		execution.StatusAt = result.FinishedAt
	}

	// Display message depending on the result
	switch {
	case result.Initialization.ErrorMessage != "":
		ui.Warn("test workflow execution failed:\n")
		ui.Errf(result.Initialization.ErrorMessage)
		return 1
	case result.IsFailed():
		ui.Warn("test workflow execution failed")
		return 1
	case result.IsAborted():
		ui.Warn("test workflow execution aborted")
		return 1
	case result.IsPassed():
		ui.Success("test workflow execution completed with success in " + result.FinishedAt.Sub(result.QueuedAt).String())
	}
	return 0
}

func uiShellGetExecution(id string) {
	ui.ShellCommand(
		"Use following command to get test workflow execution details",
		"kubectl testkube get twe "+id,
	)
}

func uiShellWatchExecution(id string) {
	ui.ShellCommand(
		"Watch test workflow execution until complete",
		"kubectl testkube watch twe "+id,
	)
}

func flattenSignatures(sig []testkube.TestWorkflowSignature) []testkube.TestWorkflowSignature {
	res := make([]testkube.TestWorkflowSignature, 0)
	for _, s := range sig {
		if len(s.Children) == 0 {
			res = append(res, s)
		} else {
			res = append(res, flattenSignatures(s.Children)...)
		}
	}
	return res
}

func printSingleResultDifference(r1 testkube.TestWorkflowStepResult, r2 testkube.TestWorkflowStepResult, signature testkube.TestWorkflowSignature, index int, steps int) bool {
	r1Status := testkube.QUEUED_TestWorkflowStepStatus
	r2Status := testkube.QUEUED_TestWorkflowStepStatus
	if r1.Status != nil {
		r1Status = *r1.Status
	}
	if r2.Status != nil {
		r2Status = *r2.Status
	}
	if r1Status == r2Status {
		return false
	}
	name := signature.Category
	if signature.Name != "" {
		name = signature.Name
	}
	took := r2.FinishedAt.Sub(r2.QueuedAt).Round(time.Millisecond)

	printStatus(signature, r2Status, took, index, steps, name, r2.ErrorMessage)
	return true
}

func printResultDifference(res1 *testkube.TestWorkflowResult, res2 *testkube.TestWorkflowResult, steps []testkube.TestWorkflowSignature) bool {
	if res1 == nil || res2 == nil {
		return false
	}
	changed := printSingleResultDifference(*res1.Initialization, *res2.Initialization, testkube.TestWorkflowSignature{Name: "Initializing"}, -1, len(steps))
	for i, s := range steps {
		changed = changed || printSingleResultDifference(res1.Steps[s.Ref], res2.Steps[s.Ref], s, i, len(steps))
	}

	return changed
}

// getTimestampLength returns length of timestamp in the line if timestamp is valid RFC timestamp.
func getTimestampLength(line string) int {
	// 29th character will be either '+' for +00:00 timestamp,
	// or 'Z' for UTC timestamp (without 00:00 section).
	if len(line) >= 29 && (line[29] == '+' || line[29] == 'Z') {
		return len(time.RFC3339Nano)
	}
	return 0
}

func printTestWorkflowLogs(signature []testkube.TestWorkflowSignature, notifications chan testkube.TestWorkflowExecutionNotification) (result *testkube.TestWorkflowResult, err error) {
	steps := flattenSignatures(signature)

	var isLineBeginning = true
	var isFirstLine = true
	for l := range notifications {
		if l.Output != nil {
			if isFirstLine {
				isFirstLine = false
			}
			continue
		}
		if l.Result != nil {
			if printResultDifference(result, l.Result, steps) {
				isLineBeginning = true
			}
			if isFirstLine {
				isFirstLine = false
			}
			result = l.Result
			continue
		}

		isLineBeginning, err = printStructuredLogLines(l.Log, isLineBeginning, isFirstLine)
		if err != nil {
			return nil, err
		}
		if isFirstLine {
			isFirstLine = false
		}
	}

	ui.NL()
	return result, nil
}

func watchTestWorkflowLogs(id string, signature []testkube.TestWorkflowSignature, client apiclientv1.Client) (result *testkube.TestWorkflowResult, err error) {
	ui.Info("Getting logs from test workflow job", id)

	// retry logic in case of error or closed channel with running state
	err = retry.Do(
		func() error {
			notifications, err := client.GetTestWorkflowExecutionNotifications(id)
			if err != nil {
				return err
			}

			// Check if result stream is closed and if execution is finished
			result, err = printTestWorkflowLogs(signature, notifications)
			if err != nil {
				return err
			}

			if result != nil && result.Status != nil &&
				(*result.Status == testkube.QUEUED_TestWorkflowStatus || *result.Status == testkube.RUNNING_TestWorkflowStatus) {
				return fmt.Errorf("test workflow execution is not finished but channel is closed")
			}

			return nil
		},
		retry.Attempts(logsRetryAttempts),
		retry.Delay(logsRetryDelay),
		retry.LastErrorOnly(true),
	)

	return result, err
}

func watchTestWorkflowServiceLogs(id, serviceName string, serviceIndex int,
	signature []testkube.TestWorkflowSignature, client apiclientv1.Client) (*testkube.TestWorkflowResult, error) {
	ui.Info("Getting logs from test workflow service job", fmt.Sprintf("%s-%s-%d", id, serviceName, serviceIndex))

	var (
		notifications chan testkube.TestWorkflowExecutionNotification
		result        *testkube.TestWorkflowResult
		nErr          error
	)

	spinner := ui.NewSpinner("Waiting for service logs")
	for {
		notifications, nErr = client.GetTestWorkflowExecutionServiceNotifications(id, serviceName, serviceIndex)
		if nErr != nil {
			execution, cErr := client.GetTestWorkflowExecution(id)
			if cErr != nil {
				spinner.Fail()
				return nil, cErr
			}

			if execution.Result != nil {
				if execution.Result.IsFinished() {
					nErr = errors.New("test workflow execution is finished")
				} else {
					time.Sleep(logsCheckDelay)
					continue
				}
			}
		}

		if nErr != nil {
			spinner.Fail()
			return nil, nErr
		}

		spinner.Stop()
		result, nErr = printTestWorkflowLogs(signature, notifications)
		if nErr != nil {
			spinner.Warning("Retrying logs")
			ui.NL()
			continue
		}

		spinner.Success("Log received")
		ui.NL()
		break
	}

	return result, nil
}

func watchTestWorkflowParallelStepLogs(id, ref string, workerIndex int,
	signature []testkube.TestWorkflowSignature, client apiclientv1.Client) (*testkube.TestWorkflowResult, error) {
	ui.Info("Getting logs from test workflow parallel step job", fmt.Sprintf("%s-%s-%d", id, ref, workerIndex))

	var (
		notifications chan testkube.TestWorkflowExecutionNotification
		result        *testkube.TestWorkflowResult
		nErr          error
	)

	spinner := ui.NewSpinner("Waiting for parallel step logs")
	for {
		notifications, nErr = client.GetTestWorkflowExecutionParallelStepNotifications(id, ref, workerIndex)
		if nErr != nil {
			execution, cErr := client.GetTestWorkflowExecution(id)
			if cErr != nil {
				spinner.Fail()
				return nil, cErr
			}

			if execution.Result != nil {
				if execution.Result.IsFinished() {
					nErr = errors.New("test workflow execution is finished")
				} else {
					time.Sleep(logsCheckDelay)
					continue
				}
			}
		}

		if nErr != nil {
			spinner.Fail()
			return nil, nErr
		}

		spinner.Stop()
		result, nErr = printTestWorkflowLogs(signature, notifications)
		if nErr != nil {
			spinner.Warning("Retrying logs")
			ui.NL()
			continue
		}

		spinner.Success("Logs received")
		ui.NL()
		break
	}

	return result, nil
}

func printStatusHeader(i, n int, name string) {
	if i == -1 {
		fmt.Println("\n" + ui.LightCyan(fmt.Sprintf("• %s", name)))
	} else {
		fmt.Println("\n" + ui.LightCyan(fmt.Sprintf("• (%d/%d) %s", i+1, n, name)))
	}
}

func printStatus(s testkube.TestWorkflowSignature, rStatus testkube.TestWorkflowStepStatus, took time.Duration,
	i, n int, name string, errorMessage string) {
	if len(errorMessage) > 0 {
		fmt.Printf("\n%s", ui.Red(errorMessage))
	}
	switch rStatus {
	case testkube.RUNNING_TestWorkflowStepStatus:
		printStatusHeader(i, n, name)
	case testkube.SKIPPED_TestWorkflowStepStatus:
		fmt.Println(ui.LightGray("• skipped"))
	case testkube.PASSED_TestWorkflowStepStatus:
		fmt.Println("\n" + ui.Green(fmt.Sprintf("• passed in %s", took)))
	case testkube.ABORTED_TestWorkflowStepStatus:
		fmt.Println("\n" + ui.Red("• aborted"))
	default:
		if s.Optional {
			fmt.Println("\n" + ui.Yellow(fmt.Sprintf("• %s in %s (ignored)", string(rStatus), took)))
		} else {
			fmt.Println("\n" + ui.Red(fmt.Sprintf("• %s in %s", string(rStatus), took)))
		}
	}
}

// if format is any RFC based timestamp
// locate next space after timestamp and trim
func trimTimestamp(line string) string {
	if strings.Index(line, "T") == 10 {
		idx := strings.Index(line, " ")
		if len(line) >= idx {
			return line[idx+1:]
		}
	}
	return line
}

func printStructuredLogLines(logs string, isLineBeginning, isFirstLine bool) (bool, error) {
	if len(logs) == 0 {
		return isLineBeginning, nil
	}
	willBeLineBeginning := logs[len(logs)-1] == '\n'
	scanner := bufio.NewScanner(strings.NewReader(logs))
	next := false
	for scanner.Scan() {
		if next {
			fmt.Print("\n")
		}
		text := trimTimestamp(scanner.Text())
		if text == registry.ErrResourceNotFound.Error() && isFirstLine {
			return isLineBeginning, registry.ErrResourceNotFound
		}
		fmt.Print(text)
		next = true
	}
	if isLineBeginning {
		fmt.Print("\n")
	}
	return willBeLineBeginning, nil
}

func printRawLogLines(logs []byte, steps []testkube.TestWorkflowSignature, execution testkube.TestWorkflowExecution) {
	currentRef := ""
	i := -1

	// Process the results
	results := make(map[string]testkube.TestWorkflowStepResult)
	if execution.Result != nil {
		if execution.Result.Steps != nil {
			results = execution.Result.Steps
		}
		if execution.Result.Initialization != nil {
			results[""] = *execution.Result.Initialization
		}
	}

	// Print error message if that's the only available thing
	if len(results) < 2 && len(logs) == 0 && len(results[""].ErrorMessage) > 0 {
		fmt.Printf("\n%s\n", ui.Red(results[""].ErrorMessage))
		return
	}

	printStatusHeader(-1, len(steps), "Initializing")

	// Strip timestamp + space for all new lines in the log
	for len(logs) > 0 {
		newLineIndex := bytes.Index(logs, NL)
		var line string
		if newLineIndex == -1 {
			line = string(logs)
			logs = nil
		} else {
			line = string(logs[:newLineIndex])
			logs = logs[newLineIndex+len(NL):]
		}

		line = trimTimestamp(line)

		start := instructions.StartHintRe.FindStringSubmatch(line)
		if len(start) == 0 {
			line += "\x07"
			fmt.Println(line)
			continue
		}

		nextRef := start[1]

		for i == -1 || (i < len(steps) && steps[i].Ref != nextRef) {
			if ps, ok := results[currentRef]; ok && ps.Status != nil {
				took := ps.FinishedAt.Sub(ps.QueuedAt).Round(time.Millisecond)
				if i != -1 {
					printStatus(steps[i], *ps.Status, took, i, len(steps), steps[i].Label(), ps.ErrorMessage)
				}
			}

			i++
			if i < len(steps) {
				currentRef = steps[i].Ref
				printStatusHeader(i, len(steps), steps[i].Label())
			}
		}
	}

	if i != -1 && i < len(steps) {
		for _, step := range steps[i:] {
			if ps, ok := results[step.Ref]; ok && ps.Status != nil {
				took := ps.FinishedAt.Sub(ps.QueuedAt).Round(time.Millisecond)
				printStatus(step, *ps.Status, took, i, len(steps), steps[i].Label(), ps.ErrorMessage)
			}

			i++
			currentRef = step.Ref
			if i < len(steps) {
				printStatusHeader(i, len(steps), steps[i].Label())
			}
		}
	}
}
