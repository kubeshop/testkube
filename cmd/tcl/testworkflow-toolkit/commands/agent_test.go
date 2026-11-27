// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	working   = cloud.AgentTurnState_AGENT_TURN_STATE_WORKING
	completed = cloud.AgentTurnState_AGENT_TURN_STATE_COMPLETED
	failed    = cloud.AgentTurnState_AGENT_TURN_STATE_FAILED
	canceled  = cloud.AgentTurnState_AGENT_TURN_STATE_CANCELED
)

// agentReply is one answer of the fake control plane.
type agentReply struct {
	turn *cloud.AgentTurn
	err  error
}

// fakeAgentTurns is a control plane that answers the get calls from a script. After the script
// it repeats a turn that still runs, and it calls onIdle one time.
type fakeAgentTurns struct {
	start   agentReply
	gets    []agentReply
	onIdle  func()
	started []*cloud.StartAgentTurnRequest
	cursors []uint32
	cancels []string
	// cancelCtxErr is the state of the context of the cancel call.
	cancelCtxErr error
}

func (f *fakeAgentTurns) StartAgentTurn(_ context.Context, req *cloud.StartAgentTurnRequest) (*cloud.AgentTurn, error) {
	f.started = append(f.started, req)
	return f.start.turn, f.start.err
}

func (f *fakeAgentTurns) GetAgentTurn(_ context.Context, _ string, after uint32) (*cloud.AgentTurn, error) {
	f.cursors = append(f.cursors, after)
	if len(f.gets) == 0 {
		if f.onIdle != nil {
			f.onIdle()
			f.onIdle = nil
		}
		return &cloud.AgentTurn{Id: "turn-1", State: working, Next: after}, nil
	}
	reply := f.gets[0]
	f.gets = f.gets[1:]
	return reply.turn, reply.err
}

func (f *fakeAgentTurns) CancelAgentTurn(ctx context.Context, turnID string) (*cloud.AgentTurn, error) {
	f.cancels = append(f.cancels, turnID)
	f.cancelCtxErr = ctx.Err()
	return &cloud.AgentTurn{Id: turnID, State: canceled}, nil
}

func textEvent(text string) *cloud.AgentTurnEvent {
	return &cloud.AgentTurnEvent{Kind: cloud.AgentTurnEventKind_AGENT_TURN_EVENT_KIND_TEXT, Text: text}
}

func toolEvent(tool, text string) *cloud.AgentTurnEvent {
	return &cloud.AgentTurnEvent{Kind: cloud.AgentTurnEventKind_AGENT_TURN_EVENT_KIND_TOOL_CALL, Tool: tool, Text: text}
}

func TestAgentStep_Run(t *testing.T) {
	unavailable := status.Error(codes.Unavailable, "no healthy upstream")
	long := strings.Repeat("é", data.MaxOutputSize)

	transientFailures := func(n int) []agentReply {
		replies := make([]agentReply, n)
		for i := range replies {
			replies[i] = agentReply{err: unavailable}
		}
		return replies
	}

	tests := []struct {
		name  string
		start agentReply
		gets  []agentReply
		// stop cancels the context of the step when the script of the control plane ends.
		stop bool
		// noStepID runs the step with no results directory.
		noStepID bool
		// outputs are the outputs that the step declares.
		outputs map[string]testworkflowsv1.ParameterSchema
		// toolOutputs is whether the control plane gives the AI Agent a tool for the outputs.
		toolOutputs bool

		wantErr     string
		wantLog     []string
		wantCursors []uint32
		wantCancels []string
		// wantOutputs is nil when the step must write no output.
		wantOutputs map[string]string
		// wantAnswer is the content of the file with the answer. Empty means no file.
		wantAnswer string
	}{
		{
			name:    "writes the declared outputs that the answer gives",
			outputs: map[string]testworkflowsv1.ParameterSchema{"verdict": verdictSchema},
			start:   agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working, SessionId: "session-1", Next: 0}},
			gets: []agentReply{
				{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "session-1", Next: 1, Answer: "All suites passed.\n" + outputsBlock(`{"verdict": "go"}`)}},
			},
			wantLog:     []string{"All suites passed.\n"},
			wantOutputs: map[string]string{"verdict": "go"},
			wantAnswer:  "All suites passed.\n" + outputsBlock(`{"verdict": "go"}`),
		},
		{
			name:        "sends no block contract and takes the outputs from the tool when the control plane has it",
			outputs:     map[string]testworkflowsv1.ParameterSchema{"verdict": verdictSchema},
			toolOutputs: true,
			start:       agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "session-1", Answer: "All suites passed.", Outputs: `{"verdict": "go"}`}},
			wantLog:     []string{"All suites passed.\n"},
			wantOutputs: map[string]string{"verdict": "go"},
			wantAnswer:  "All suites passed.",
		},
		{
			name:        "takes the outputs of a turn that ended on the tool and gave no text",
			outputs:     map[string]testworkflowsv1.ParameterSchema{"verdict": verdictSchema},
			toolOutputs: true,
			start:       agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "session-1", Outputs: `{"verdict": "go"}`}},
			wantOutputs: map[string]string{"verdict": "go"},
		},
		{
			name:    "fails for an answer with no valid outputs, and writes the answer and the defaults",
			outputs: map[string]testworkflowsv1.ParameterSchema{"verdict": verdictSchema},
			start:   agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working, SessionId: "session-1", Next: 0}},
			gets: []agentReply{
				{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "session-1", Next: 1, Answer: "Go."}},
			},
			wantErr:     "the answer of the AI Agent does not give the outputs of the step: the answer has no `testkube-outputs` block",
			wantOutputs: map[string]string{"verdict": "no-go"},
			wantAnswer:  "Go.",
		},
		{
			name:        "writes the defaults when the control plane refuses the turn",
			outputs:     map[string]testworkflowsv1.ParameterSchema{"verdict": verdictSchema},
			start:       agentReply{err: status.Error(codes.ResourceExhausted, "the execution started 20 turns")},
			wantErr:     "cannot start the AI Agent",
			wantLog:     []string{"warn: the AI Agent gave no valid outputs, so the step uses the defaults: verdict=no-go\n"},
			wantOutputs: map[string]string{"verdict": "no-go"},
		},
		{
			name:  "prints the events and writes the answer of a completed turn to the file",
			start: agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working, SessionId: "session-1", Next: 0}},
			gets: []agentReply{
				{turn: &cloud.AgentTurn{Id: "turn-1", State: working, Next: 2, Events: []*cloud.AgentTurnEvent{
					textEvent("I read the logs."), toolEvent("fetch_execution_logs", `{"executionId":"e1"}`),
				}}},
				{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "session-1", Next: 3, Answer: "  The database was down.\n",
					Events: []*cloud.AgentTurnEvent{textEvent("The database was down.")}}},
			},
			wantLog:     []string{"I read the logs.\n", "Tool call: fetch_execution_logs {\"executionId\":\"e1\"}\n", "The database was down.\n"},
			wantCursors: []uint32{0, 2},
			wantAnswer:  "  The database was down.\n",
		},
		{
			name:       "uses a turn that is completed in the answer to the start call, and prints an answer that no event gave",
			start:      agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "session-1", Answer: "pass"}},
			wantLog:    []string{"Session: session-1\npass\n"},
			wantAnswer: "pass",
		},
		{
			name:       "writes an answer above the size of a step output to the file in full",
			start:      agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "session-1", Answer: long}},
			wantAnswer: long,
		},
		{
			name: "removes the instruction start byte from the log, the output and the file",
			start: agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "s\x01\x051", Answer: "ok\n\x01\x01\x05\x05;output",
				Events: []*cloud.AgentTurnEvent{textEvent("\x01\x05;pass"), toolEvent("\x01\x05tool", "\x01\x05args")}}},
			wantLog:    []string{"\x05;pass\n", "Tool call: \x05tool \x05args\n"},
			wantAnswer: "ok\n\x05\x05;output",
		},
		{
			name:     "starts no turn for a step with no ID",
			start:    agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, SessionId: "session-1", Answer: "pass"}},
			noStepID: true,
			wantErr:  "the step has no results directory for the answer",
		},
		{
			name:    "fails when the start call fails",
			start:   agentReply{err: status.Error(codes.NotFound, "AI Agent not found")},
			wantErr: `cannot start the AI Agent "failure-triage"`,
		},
		{
			name:    "fails with the message of a failed turn and keeps its session",
			start:   agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working, SessionId: "session-1"}},
			gets:    []agentReply{{turn: &cloud.AgentTurn{Id: "turn-1", State: failed, Message: "The tool run_workflow\x01 needs an approval."}}},
			wantErr: "the AI Agent failed: The tool run_workflow needs an approval.",
		},
		{
			name:    "fails with the message of a canceled turn",
			start:   agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working}},
			gets:    []agentReply{{turn: &cloud.AgentTurn{Id: "turn-1", State: canceled, Message: "The execution ended."}}},
			wantErr: "was canceled: The execution ended.",
		},
		{
			name:        "cancels a turn in a state that it does not know",
			start:       agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: cloud.AgentTurnState(99)}},
			wantErr:     "which this Runner does not know",
			wantCancels: []string{"turn-1"},
		},
		{
			name:        "continues after transient read errors below the limit",
			start:       agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working, Next: 4}},
			gets:        append(transientFailures(agentPollFailureLimit-1), agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: completed, Answer: "pass"}}),
			wantCursors: repeatCursor(4, agentPollFailureLimit),
			wantAnswer:  "pass",
		},
		{
			name:        "cancels the turn one time at the limit of transient read errors",
			start:       agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working}},
			gets:        transientFailures(agentPollFailureLimit),
			wantErr:     "cannot read the turn of the AI Agent",
			wantCancels: []string{"turn-1"},
		},
		{
			name:        "cancels the turn one time at a read error that is not transient",
			start:       agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working}},
			gets:        []agentReply{{err: status.Error(codes.PermissionDenied, "the execution ended")}},
			wantErr:     "cannot read the turn of the AI Agent",
			wantCursors: []uint32{0},
			wantCancels: []string{"turn-1"},
		},
		{
			name:        "cancels the turn one time when the step stops",
			start:       agentReply{turn: &cloud.AgentTurn{Id: "turn-1", State: working}},
			stop:        true,
			wantErr:     "the step stopped before the AI Agent gave its answer",
			wantCancels: []string{"turn-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, stop := context.WithCancelCause(context.Background())
			t.Cleanup(func() { stop(nil) })

			client := &fakeAgentTurns{start: tt.start, gets: tt.gets}
			if tt.stop {
				client.onIdle = func() { stop(errors.New("received signal: terminated")) }
			}
			var log bytes.Buffer
			dir := t.TempDir()
			step := &agentStep{
				client:       client,
				out:          &log,
				requestID:    "request-1",
				stepRef:      "ref-1",
				outputsDir:   filepath.Join(dir, "outputs"),
				resultsDir:   filepath.Join(dir, "results"),
				pollInterval: time.Millisecond,
				toolOutputs:  tt.toolOutputs,
			}
			if tt.noStepID {
				step.resultsDir = ""
			}

			err := step.run(ctx, testworkflowsv1.StepAgent{Name: "failure-triage", Prompt: "Find the cause.", Outputs: tt.outputs})

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.NotContains(t, err.Error(), "\x01")
			} else {
				require.NoError(t, err)
			}

			if tt.noStepID {
				assert.Empty(t, client.started)
			} else {
				require.Len(t, client.started, 1)
				assert.Equal(t, "request-1", client.started[0].GetRequestId())
				assert.Equal(t, "ref-1", client.started[0].GetStepRef())
				assert.Equal(t, "failure-triage", client.started[0].GetAgentName())
				wantPrompt := "Find the cause."
				if !tt.toolOutputs {
					wantPrompt += agentOutputsContract(tt.outputs)
				}
				assert.Equal(t, wantPrompt, client.started[0].GetPrompt())
				wantSchema, err := agentOutputsSchemaJSON(tt.outputs)
				require.NoError(t, err)
				assert.Equal(t, wantSchema, client.started[0].GetOutputsSchema())
			}

			for _, line := range tt.wantLog {
				assert.Contains(t, log.String(), line)
			}
			assert.NotContains(t, log.String(), "\x01")
			// The log is what the AI Agent returned: the step adds no line about the file or the outputs.
			assert.NotContains(t, log.String(), "The answer is in")
			assert.NotContains(t, log.String(), "Output ")
			if tt.wantAnswer != "" {
				assert.Equal(t, 1, strings.Count(log.String(), strings.TrimSpace(tt.wantAnswer)), "the log has the answer one time")
			}
			if tt.wantCursors != nil {
				assert.Equal(t, tt.wantCursors, client.cursors)
			}
			assert.Equal(t, tt.wantCancels, client.cancels)
			assert.NoError(t, client.cancelCtxErr, "the cancel call must not use the context of the step")

			assert.Equal(t, tt.wantOutputs, readDir(t, step.outputsDir))
			files := readDir(t, filepath.Join(dir, "results"))
			if session := wantAgentSession(tt.noStepID, tt.start, tt.gets); session != "" {
				var info map[string]string
				require.NoError(t, json.Unmarshal([]byte(files[data.StepAgentFile]), &info))
				assert.Equal(t, map[string]string{data.StepAgentSession: session, data.StepAgentName: "failure-triage"}, info)
			} else {
				assert.NotContains(t, files, data.StepAgentFile)
			}
			delete(files, data.StepAgentFile)
			wantFiles := map[string]string{}
			if tt.wantAnswer != "" {
				wantFiles[agentResponseFile] = tt.wantAnswer
			}
			if files == nil {
				files = map[string]string{}
			}
			assert.Equal(t, wantFiles, files)
		})
	}
}

// wantAgentSession returns the session that the step must write for step.<id>.agent.session: the
// first one that the control plane gives, with no instruction start byte.
func wantAgentSession(noStepID bool, start agentReply, gets []agentReply) string {
	if noStepID {
		return ""
	}
	for _, reply := range append([]agentReply{start}, gets...) {
		if reply.turn != nil && reply.turn.GetSessionId() != "" {
			return strings.ReplaceAll(reply.turn.GetSessionId(), "\x01", "")
		}
	}
	return ""
}

func repeatCursor(cursor uint32, n int) []uint32 {
	cursors := make([]uint32, n)
	for i := range cursors {
		cursors[i] = cursor
	}
	return cursors
}

// readDir returns the content of each file in the directory, and nil when it does not exist.
func readDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	files := make(map[string]string)
	for _, entry := range entries {
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err)
		files[entry.Name()] = string(content)
	}
	return files
}

func TestPlainText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "leaves normal text as it is", in: "The test passed.\n", want: "The test passed.\n"},
		{name: "removes the start of an instruction", in: "\x01\x05;pass", want: "\x05;pass"},
		{name: "leaves no new pair when it removes a pair inside a pair", in: "\x01\x01\x05\x05", want: "\x05\x05"},
		{name: "removes a byte that stands alone", in: "a\x01b", want: "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plainText(tt.in)

			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "\x01")
		})
	}
}

func TestAgentCapabilityError(t *testing.T) {
	enabled := func(name capabilities.Capability, on bool) *cloud.Capability {
		return &cloud.Capability{Name: string(name), Enabled: on}
	}

	tests := []struct {
		name      string
		published []*cloud.Capability
		wantErr   bool
	}{
		{name: "the capability is on", published: []*cloud.Capability{enabled(capabilities.CapabilityAgentStep, true)}},
		{name: "the control plane publishes other capabilities only", published: []*cloud.Capability{enabled(capabilities.CapabilityCloudStorage, true)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agentCapabilityError(tt.published)

			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), string(capabilities.CapabilityAgentStep))
			assert.Contains(t, err.Error(), "Upgrade the control plane")
		})
	}
}

func TestResolveAgentSpec(t *testing.T) {
	machine := expressions.NewMachine().
		Register("execution.id", "exec-1").
		Register("step.tests.outputs.summary", "3 of 40 failed, see {{ execution.id }}").
		Register("agent", "failure-triage")

	tests := []struct {
		name    string
		spec    testworkflowsv1.StepAgent
		want    testworkflowsv1.StepAgent
		wantErr string
	}{
		{
			name: "resolves the name and the prompt",
			spec: testworkflowsv1.StepAgent{Name: "{{ agent }}", Prompt: "Find the cause in {{ execution.id }}."},
			want: testworkflowsv1.StepAgent{Name: "failure-triage", Prompt: "Find the cause in exec-1."},
		},
		{
			name: "does not resolve an expression in the value of a step output",
			spec: testworkflowsv1.StepAgent{Name: "failure-triage", Prompt: "{{ step.tests.outputs.summary }}"},
			want: testworkflowsv1.StepAgent{Name: "failure-triage", Prompt: "3 of 40 failed, see {{ execution.id }}"},
		},
		{
			name: "accepts an empty prompt",
			spec: testworkflowsv1.StepAgent{Name: "failure-triage"},
			want: testworkflowsv1.StepAgent{Name: "failure-triage"},
		},
		{
			name:    "fails on an output that does not exist",
			spec:    testworkflowsv1.StepAgent{Name: "failure-triage", Prompt: "{{ step.missing.outputs.summary }}"},
			wantErr: "The agent step is invalid",
		},
		{
			name:    "fails on a name that resolves to nothing",
			spec:    testworkflowsv1.StepAgent{Name: `{{ "" }}`},
			wantErr: "the name of the AI Agent is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveAgentSpec(tt.spec, machine)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				var definitionErr *DefinitionError
				assert.ErrorAs(t, err, &definitionErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAgentMachine(t *testing.T) {
	registry := executiondata.NewRegistry()
	registry.Add(executiondata.Execution{
		Id:       "exec-2",
		Workflow: "tests",
		Outputs:  map[string]string{"summary": "3 of 40 failed"},
	})
	machine := agentMachine(executiondata.NewMachine(executiondata.MachineOptions{Registry: registry}))

	tests := []struct {
		name    string
		prompt  string
		want    string
		wantErr string
	}{
		{
			name:   "reads an output of an execution that an earlier step started",
			prompt: `Find the cause: {{ execution("tests").outputs.summary }}`,
			want:   "Find the cause: 3 of 40 failed",
		},
		{
			name:    "fails on an execution that no step started",
			prompt:  `{{ execution("other").outputs.summary }}`,
			wantErr: `unknown execution "other"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveAgentSpec(testworkflowsv1.StepAgent{Name: "failure-triage", Prompt: tt.prompt}, machine)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Prompt)
		})
	}
}
