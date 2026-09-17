package commands

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/obfuscator"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runtime"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

const (
	// maxStepErrorSize is the limit for the step message that a toolkit step writes. The message is one line, not a log.
	maxStepErrorSize = 1024
	// maxStepErrorReadSize limits the read of the step message file. It is larger than maxStepErrorSize,
	// so the masking also finds a sensitive value that crosses the limit of the message.
	maxStepErrorReadSize = 64 * 1024
	// stepErrorMask replaces a sensitive value in the step message.
	stepErrorMask = "*****"
)

// ansiEscapeRe matches an ANSI escape sequence, also a sequence that the size limit cuts.
var ansiEscapeRe = regexp.MustCompile(`\x1b(\[[0-9;]*[A-Za-z]?)?`)

func Run(ctx context.Context, run lite.ActionExecute, container lite.LiteActionContainer) {
	machine := runtime.GetInternalTestWorkflowMachine()
	state := data.GetState()
	step := state.GetStep(run.Ref)

	// Abandon executing if the step was finished before
	if step.IsFinished() {
		return
	}

	// Obtain command to run
	command := make([]string, 0)
	if container.Config.Command != nil {
		command = slices.Clone(*container.Config.Command)
	}
	if container.Config.Args != nil {
		command = append(command, *container.Config.Args...)
	}

	// Ensure the command is not empty
	if len(command) == 0 {
		output.ExitErrorf(constants.CodeInputError, "command is required")
	}

	// Resolve the command to run
	expandedCommand := make([]string, 0, len(command))
	for i := range command {
		// Check if this argument is a pure template expression that may resolve to an array
		if innerExpr, isPure := expressions.ExtractPureTemplateExpression(command[i]); isPure && !expressions.IsWildcardAccessorOnly(innerExpr) {
			expr, err := expressions.CompileAndResolve(innerExpr, machine, expressions.FinalizerFail)
			if err != nil {
				output.ExitErrorf(constants.CodeInternal, "failed to compute argument '%d': %s", i, err.Error())
			}
			if expr.Static() != nil {
				// Array result: expand into individual arguments
				if items, sliceErr := expr.Static().SliceValue(); sliceErr == nil {
					for _, item := range items {
						sv := expressions.NewValue(item)
						s, _ := sv.StringValue()
						expandedCommand = append(expandedCommand, s)
					}
					continue
				}
				// Non-array result: reuse the already-resolved value
				s, _ := expr.Static().StringValue()
				expandedCommand = append(expandedCommand, s)
				continue
			}
		}
		value, err := expressions.CompileAndResolveTemplate(command[i], machine, expressions.FinalizerFail)
		if err != nil {
			output.ExitErrorf(constants.CodeInternal, "failed to compute argument '%d': %s", i, err.Error())
		}
		s, _ := value.Static().StringValue()
		expandedCommand = append(expandedCommand, s)
	}
	command = expandedCommand

	// Ensure the command is not empty after expansion
	if len(command) == 0 {
		output.ExitErrorf(constants.CodeInputError, "command is required")
	}

	// An output another workflow withheld resolves to a marker instead of the value it
	// was meant to carry. Running the command would hand the marker to the tool as if
	// it were that value, so fail while the cause is still visible.
	if markers := executiondata.WithheldMarkersIn(command); len(markers) > 0 {
		output.ExitErrorf(constants.CodeInputError, "%s", executiondata.WithheldError("the command of this step", markers).Error())
	}

	// Remove the message of an earlier step or attempt. When the file stays, its message can be old, so the step does not read it.
	// Both files must be gone before the step runs, so a message of an earlier attempt does not
	// reach this one.
	stepErrorFresh := removeStepError(constants.StepErrorPath) && removeStepError(constants.StepReasonPath)

	// Run the operation with context
	execution := orchestration.Executions.CreateWithContext(ctx, command[0], command[1:])
	result, err := execution.Run()
	if err != nil {
		output.ExitErrorf(constants.CodeInternal, "failed to execute: %v", err)
	}

	// Initialize local state
	var status constants.StepStatus

	success := result.ExitCode == 0

	// Compute the result
	if run.Negative {
		success = !success
	}
	if result.Aborted {
		status = constants.StepStatusAborted
	} else if success {
		status = constants.StepStatusPassed
	} else {
		status = constants.StepStatusFailed
	}

	// Abandon saving execution data if the step has been finished before
	if step.IsFinished() {
		// The timeout handler sets the status before it kills the process, so the step sends the timeout cause here, not after the exit.
		if *step.Status == constants.StepStatusTimeout {
			orchestration.FinishTimedOutExecution(step)
		}
		return
	}

	// An aborted step keeps the cause of the abort, so only a failed step reads the file.
	details, reason := result.Details, result.Reason
	if run.Toolkit && status == constants.StepStatusFailed && stepErrorFresh {
		var sensitiveValues []string
		if orchestration.Setup != nil {
			sensitiveValues = orchestration.Setup.GetSensitiveValues()
		}
		// A toolkit step writes the words of its failure, and a code next to them when it has one.
		details, reason = readStepError(constants.StepErrorPath, sensitiveValues), readStepReason(constants.StepReasonPath)
	}

	// Notify about the status
	step.SetStatus(status).SetExitCode(result.ExitCode)
	orchestration.FinishExecution(step, constants.ExecutionResult{ExitCode: result.ExitCode, Details: details, Reason: reason, Iteration: int(step.Iteration)})
}

// removeStepError removes the step message file. It returns false when the file is still there.
func removeStepError(path string) bool {
	err := os.Remove(path)
	return err == nil || errors.Is(err, os.ErrNotExist)
}

// readStepReason returns the first line of the step reason file, which holds a reason code.
// A code is short and holds no sensitive value, so it needs no masking.
func readStepReason(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	reason := string(content)
	if i := strings.IndexAny(reason, "\r\n"); i >= 0 {
		reason = reason[:i]
	}
	return strings.TrimSpace(reason)
}

// readStepError returns the first line of the step message file as plain text, with a size limit.
// It masks the sensitive values before it cuts the text, because a part of a sensitive value does not match the masking.
func readStepError(path string, sensitiveValues []string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, maxStepErrorReadSize))
	if err != nil {
		return ""
	}
	message := maskSensitiveValues(ansiEscapeRe.ReplaceAllString(string(content), ""), sensitiveValues)
	if i := strings.IndexAny(message, "\r\n"); i >= 0 {
		message = message[:i]
	}
	if len(message) > maxStepErrorSize {
		message = message[:maxStepErrorSize]
	}
	// The limit can cut a multibyte character, so remove an incomplete character at the end.
	return strings.TrimSpace(strings.ToValidUTF8(message, ""))
}

// maskSensitiveValues replaces each sensitive value in the text.
func maskSensitiveValues(text string, sensitiveValues []string) string {
	if text == "" || len(sensitiveValues) == 0 {
		return text
	}
	var buf bytes.Buffer
	o := obfuscator.New(&buf, obfuscator.FullReplace(stepErrorMask), sensitiveValues)
	_, _ = o.Write([]byte(text))
	_ = o.Flush()
	return buf.String()
}
