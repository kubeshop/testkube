// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	toolkitcommon "github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/grpcutils"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

// agentSubject names the agent step in an error of its definition.
const agentSubject = "The agent step"

// agentMinControlPlane is the first control plane version that can run the agent step.
const agentMinControlPlane = "2.16.0"

const (
	agentPollInterval = 2 * time.Second
	// agentCallTimeout limits one call, so a connection that hangs counts as a failed call.
	agentCallTimeout = time.Minute
	// agentCancelTimeout limits the cancel call. The process has little time after a stop
	// signal, and the control plane also cancels a turn that nobody reads.
	agentCancelTimeout = 5 * time.Second
	// agentPollFailureLimit is the number of failed reads in sequence that the step accepts.
	// A restart of the control plane must not fail a turn that still runs there.
	agentPollFailureLimit = 10
)

const (
	agentOutputSession = "session"
	// agentResponseFile is the file in the step results directory that holds the answer. The
	// answer is not a step output, because a step output has a size limit and an answer has none.
	agentResponseFile = "response.md"
)

// instructionStart is the first byte of an instruction for the init process and for the Runner.
const instructionStart = "\x01"

// plainText removes each instruction start byte from text that an AI Agent or a tool wrote.
//
// A log line that starts with this byte and the byte 0x05 is an instruction, and an instruction
// can set the result or an output of a different step. Removing only the pair is not sufficient,
// because the text around a removed pair can join into a new pair.
func plainText(s string) string {
	return strings.ReplaceAll(s, instructionStart, "")
}

// agentCapabilityError returns an error when the control plane cannot run the agent step.
func agentCapabilityError(published []*cloud.Capability) error {
	if capabilities.Enabled(published, capabilities.CapabilityAgentStep) {
		return nil
	}
	return fmt.Errorf("this control plane cannot run the agent step: it does not publish the %q capability. "+
		"The agent step needs control plane %s or later, with AI Agent steps enabled",
		string(capabilities.CapabilityAgentStep), agentMinControlPlane)
}

// resolveAgentSpec resolves the expressions in the spec of the step. The processor hides the spec
// from the init process, so this is the only place that resolves them.
func resolveAgentSpec(spec testworkflowsv1.StepAgent, machines ...expressions.Machine) (testworkflowsv1.StepAgent, error) {
	if err := expressions.Finalize(&spec, machines...); err != nil {
		return spec, &DefinitionError{Subject: agentSubject, Err: err}
	}
	if spec.Name == "" {
		return spec, &DefinitionError{Subject: agentSubject, Err: errors.New("the name of the AI Agent is empty")}
	}
	return spec, nil
}

// agentStep runs one turn of an AI Agent and gives its answer to the next steps.
type agentStep struct {
	client controlplaneclient.AgentTurnClient
	out    io.Writer
	// requestID identifies this attempt of the step. Each start call of the process sends it, so
	// a retry of the call returns the turn that the first call started.
	requestID string
	stepRef   string
	// outputsDir is the directory that the init process reads the step outputs from.
	outputsDir string
	// resultsDir is the results directory of the step. It is empty for a step with no ID, and
	// then the answer is only in the step log.
	resultsDir   string
	pollInterval time.Duration
}

// run starts the turn and follows it to its end. It returns an error for each end that is not a
// completed turn.
func (s *agentStep) run(ctx context.Context, spec testworkflowsv1.StepAgent) error {
	turn, err := s.call(ctx, func(ctx context.Context) (*cloud.AgentTurn, error) {
		return s.client.StartAgentTurn(ctx, &cloud.StartAgentTurnRequest{
			RequestId: s.requestID,
			StepRef:   s.stepRef,
			AgentName: spec.Name,
			Prompt:    spec.Prompt,
		})
	})
	if err != nil {
		return fmt.Errorf("cannot start the AI Agent %q: %w", spec.Name, err)
	}
	turnID := turn.GetId()
	s.printf("AI Agent %q started. Session: %s\n", spec.Name, plainText(turn.GetSessionId()))

	for {
		s.printEvents(turn.GetEvents())

		switch turn.GetState() {
		case cloud.AgentTurnState_AGENT_TURN_STATE_COMPLETED:
			return s.writeOutputs(turn)
		case cloud.AgentTurnState_AGENT_TURN_STATE_FAILED:
			return fmt.Errorf("the AI Agent failed: %s", plainText(turn.GetMessage()))
		case cloud.AgentTurnState_AGENT_TURN_STATE_CANCELED:
			return fmt.Errorf("the turn of the AI Agent was canceled: %s", plainText(turn.GetMessage()))
		case cloud.AgentTurnState_AGENT_TURN_STATE_WORKING:
		default:
			// A state that this toolkit does not know can be a state that waits for input. The
			// step cannot give input, so it must not wait for the end of such a turn.
			s.cancel(turnID)
			return fmt.Errorf("the control plane returned the turn state %q, which this Runner does not know. Upgrade the Runner", turn.GetState().String())
		}

		turn, err = s.poll(ctx, turnID, turn.GetNext())
		if err != nil {
			// The step ends here, and the turn can still run in the control plane.
			s.cancel(turnID)
			return err
		}
	}
}

// poll waits one poll interval and reads the turn. After a transient error it waits and reads
// again, up to the failure limit.
func (s *agentStep) poll(ctx context.Context, turnID string, after uint32) (*cloud.AgentTurn, error) {
	for failures := 0; ; {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("the step stopped before the AI Agent gave its answer: %w", context.Cause(ctx))
		case <-time.After(s.pollInterval):
		}

		turn, err := s.call(ctx, func(ctx context.Context) (*cloud.AgentTurn, error) {
			return s.client.GetAgentTurn(ctx, turnID, after)
		})
		if err == nil {
			return turn, nil
		}
		if ctx.Err() != nil {
			continue
		}
		failures++
		if !isTransientCallError(err) || failures >= agentPollFailureLimit {
			return nil, fmt.Errorf("cannot read the turn of the AI Agent: %w", err)
		}
	}
}

// call runs one call to the control plane with the call timeout.
func (s *agentStep) call(ctx context.Context, fn func(ctx context.Context) (*cloud.AgentTurn, error)) (*cloud.AgentTurn, error) {
	ctx, cancel := context.WithTimeout(ctx, agentCallTimeout)
	defer cancel()
	return fn(ctx)
}

// isTransientCallError reports whether a later call can succeed after this error.
func isTransientCallError(err error) bool {
	return grpcutils.IsTransient(err) || errors.Is(err, context.DeadlineExceeded)
}

// cancel asks the control plane to stop the turn. It runs when the step ends before the turn
// does, so the context of the step can be done already and the call uses its own.
func (s *agentStep) cancel(turnID string) {
	ctx, cancel := context.WithTimeout(context.Background(), agentCancelTimeout)
	defer cancel()
	if _, err := s.client.CancelAgentTurn(ctx, turnID); err != nil {
		s.printf("warn: cannot cancel the turn of the AI Agent: %s\n", err.Error())
	}
}

func (s *agentStep) printEvents(events []*cloud.AgentTurnEvent) {
	for _, event := range events {
		text := plainText(event.GetText())
		switch event.GetKind() {
		case cloud.AgentTurnEventKind_AGENT_TURN_EVENT_KIND_TOOL_CALL:
			line := "Tool call: " + plainText(event.GetTool())
			if text != "" {
				line += " " + text
			}
			s.printf("%s\n", line)
		default:
			if text != "" {
				s.printf("%s\n", text)
			}
		}
	}
}

// writeOutputs writes the file with the answer and the session output.
func (s *agentStep) writeOutputs(turn *cloud.AgentTurn) error {
	if s.resultsDir == "" {
		s.printf("The step has no ID, so the next steps cannot read the answer.\n")
	} else {
		path := filepath.Join(s.resultsDir, agentResponseFile)
		if err := writeStepFile(path, plainText(turn.GetAnswer())); err != nil {
			return fmt.Errorf("cannot write the answer of the AI Agent: %w", err)
		}
		s.printf("The answer is in %s\n", path)
	}
	if err := writeStepFile(filepath.Join(s.outputsDir, agentOutputSession), plainText(turn.GetSessionId())); err != nil {
		return fmt.Errorf("cannot write the %q output: %w", agentOutputSession, err)
	}
	return nil
}

func (s *agentStep) printf(format string, args ...any) {
	fmt.Fprintf(s.out, format, args...)
}

// writeStepFile writes a file that the next steps read. They can run as a different user.
func writeStepFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0666)
}

func NewAgentCmd() *cobra.Command {
	var base64Encoded bool

	cmd := &cobra.Command{
		Use:   "agent <spec>",
		Short: "Run an AI Agent",
		Args:  cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			if !base64Encoded {
				toolkitcommon.Fail(&DefinitionError{Subject: agentSubject, Err: errors.New("the spec must be base64 encoded")})
			}
			var spec testworkflowsv1.StepAgent
			if err := expressionstcl.DecodeBase64JSON(args[0], &spec); err != nil {
				toolkitcommon.Fail(&DefinitionError{Subject: agentSubject, Err: err})
			}

			if err := agentCapabilityError(env.GetCapabilities()); err != nil {
				toolkitcommon.Fail(err)
			}

			machine := expressions.CombinedMachines(data.GetBaseTestWorkflowMachine(), data.ExecutionMachine())
			spec, err := resolveAgentSpec(spec, machine)
			if err != nil {
				toolkitcommon.Fail(err)
			}

			client, err := env.Cloud()
			if err != nil {
				toolkitcommon.Fail(fmt.Errorf("cannot reach the control plane: %w", err))
			}

			// The expression has no value for a step with no ID.
			resultsDir, _ := expressions.EvalExpression("step.results", machine)
			step := &agentStep{
				client:       client,
				out:          os.Stdout,
				requestID:    uuid.NewString(),
				stepRef:      config.Ref(),
				outputsDir:   data.GetOutputsDir(),
				pollInterval: agentPollInterval,
			}
			if resultsDir != nil {
				step.resultsDir, _ = resultsDir.StringValue()
			}

			if err := step.run(cmd.Context(), spec); err != nil {
				toolkitcommon.Fail(err)
			}
		},
	}

	cmd.Flags().BoolVar(&base64Encoded, "base64", false, "spec is base64 encoded")

	return cmd
}
