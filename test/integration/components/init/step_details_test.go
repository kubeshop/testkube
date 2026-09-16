package init_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runner"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestInitProcessStepDetails_Integration(t *testing.T) {
	test.IntegrationTest(t)

	// run returns the actions that start, execute, and end a declared step.
	run := func(ref, script string, toolkit, negative bool) []map[string]any {
		return []map[string]any{
			{"S": ref},
			{"c": map[string]any{"r": ref, "c": map[string]any{"command": []string{script}}}},
			{"e": map[string]any{"r": ref, "t": toolkit, "n": negative}},
			{"E": ref},
		}
	}

	// step returns the actions that run one script as a step, with an optional timeout.
	step := func(ref, script, timeout string, toolkit, negative bool, parents ...string) []map[string]any {
		actions := []map[string]any{{"d": map[string]any{"c": "true", "r": ref, "p": parents}}}
		if timeout != "" {
			actions = append(actions, map[string]any{"t": map[string]any{"r": ref, "t": timeout}})
		}
		return append(actions, run(ref, script, toolkit, negative)...)
	}

	tests := []struct {
		name string
		// groups returns the action groups after the setup group. The test reads the result of the step "step".
		groups func(script string) [][]map[string]any
		script string
		pause  time.Duration
		// staleError is the content of a step message file that exists before the step starts.
		staleError   string
		wantDetails  string
		wantExitCode uint8
	}{
		{
			name:   "a step with a retry that SIGKILL stops keeps the process-killed message and does not retry",
			script: "#!/bin/sh\nkill -9 $$\n",
			groups: func(script string) [][]map[string]any {
				declare := []map[string]any{
					{"d": map[string]any{"c": "true", "r": "step"}},
					{"R": map[string]any{"r": "step", "c": 2}},
				}
				return [][]map[string]any{append(declare, run("step", script, false, false)...)}
			},
			wantDetails:  "the test process was killed, possibly by an out-of-memory kill (signal: killed)",
			wantExitCode: constants.CodeAborted,
		},
		{
			name:   "a step that runs longer than its timeout reports the step-timeout message",
			script: "#!/bin/sh\nsleep 5\n",
			groups: func(script string) [][]map[string]any {
				return [][]map[string]any{step("step", script, "1s", false, false)}
			},
			wantDetails:  "the step did not finish within its timeout",
			wantExitCode: constants.CodeAborted,
		},
		{
			name:   "a step whose group timeout ended before the step started reports the step-timeout message",
			script: "#!/bin/sh\nexit 0\n",
			groups: func(script string) [][]map[string]any {
				first := append([]map[string]any{
					{"d": map[string]any{"c": "true", "r": "group"}},
					{"t": map[string]any{"r": "group", "t": "1s"}},
					{"S": "group"},
				}, step("first", script, "", false, false, "group")...)
				second := append(step("step", script, "", false, false, "group"), map[string]any{"E": "group"})
				return [][]map[string]any{first, second}
			},
			// The second container group starts after the group timeout ended.
			pause:        1500 * time.Millisecond,
			wantDetails:  "the step did not finish within its timeout",
			wantExitCode: constants.CodeAborted,
		},
		{
			name:   "a failed toolkit step reports the message of the file",
			script: "#!/bin/sh\nprintf 'toolkit cause' > \"$TK_ERR_FILE\"\nexit 1\n",
			groups: func(script string) [][]map[string]any {
				return [][]map[string]any{step("step", script, "", true, false)}
			},
			wantDetails:  "toolkit cause",
			wantExitCode: 1,
		},
		{
			name:   "a toolkit step killed by SIGKILL reports the process-killed message instead of the file",
			script: "#!/bin/sh\nprintf 'toolkit cause' > \"$TK_ERR_FILE\"\nkill -9 $$\n",
			groups: func(script string) [][]map[string]any {
				return [][]map[string]any{step("step", script, "", true, false)}
			},
			wantDetails:  "the test process was killed, possibly by an out-of-memory kill (signal: killed)",
			wantExitCode: constants.CodeAborted,
		},
		{
			name:   "a negative toolkit step that passes reports no message",
			script: "#!/bin/sh\nprintf 'toolkit cause' > \"$TK_ERR_FILE\"\nexit 1\n",
			groups: func(script string) [][]map[string]any {
				return [][]map[string]any{step("step", script, "", true, true)}
			},
			wantDetails:  "",
			wantExitCode: 1,
		},
		{
			name:   "a failed step that is not a toolkit step does not report the file",
			script: "#!/bin/sh\nprintf 'user cause' > \"$TESTKUBE_TW_INTERNAL_PATH/error\"\nexit 1\n",
			groups: func(script string) [][]map[string]any {
				return [][]map[string]any{step("step", script, "", false, false)}
			},
			wantDetails:  "",
			wantExitCode: 1,
		},
		{
			name:   "a failed toolkit step does not report the message of an earlier step",
			script: "#!/bin/sh\nexit 1\n",
			groups: func(script string) [][]map[string]any {
				return [][]map[string]any{step("step", script, "", true, false)}
			},
			staleError:   "earlier cause",
			wantDetails:  "",
			wantExitCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(testDir, ".tktw"), 0755))
			require.NoError(t, os.WriteFile(filepath.Join(testDir, "termination.log"), []byte{}, 0666))
			script := filepath.Join(testDir, "step.sh")
			require.NoError(t, os.WriteFile(script, []byte(tt.script), 0755))
			if tt.staleError != "" {
				require.NoError(t, os.WriteFile(filepath.Join(testDir, ".tktw", "error"), []byte(tt.staleError), 0666))
			}

			groups := tt.groups(script)
			actions := append([][]map[string]any{{{"_": map[string]bool{"i": true, "t": true, "b": true}}}}, groups...)
			setupEnvWithActions(t, testDir, actions)
			updateConstants(testDir)

			initializeOrchestration(t)
			t.Cleanup(func() { cleanupOrchestration(t) })
			exitCode, err := runner.RunInit(0)
			require.NoError(t, err)
			require.Equal(t, 0, exitCode)

			var out []byte
			for i := 1; i <= len(groups); i++ {
				if i > 1 {
					time.Sleep(tt.pause)
				}
				initializeOrchestration(t)
				out = captureStdout(t, func() {
					_, err = runner.RunInit(i)
				})
				require.NoError(t, err)
			}

			result, ok := findExecutionResult(t, out, "step")
			require.True(t, ok, "the init process did not send the execution result of the step:\n%s", out)
			assert.Equal(t, tt.wantDetails, result.Details)
			assert.Equal(t, tt.wantExitCode, result.ExitCode)
			assert.Zero(t, data.GetState().GetStep("step").Iteration, "the step started another attempt")
		})
	}
}

// captureStdout collects what fn prints to the standard output. The init process prints the hints there.
func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original })

	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()
	fn()
	os.Stdout = original
	require.NoError(t, w.Close())
	return <-done
}

// findExecutionResult returns the last execution result of the step, because the notifier keeps the result of the last attempt.
func findExecutionResult(t *testing.T, out []byte, ref string) (constants.ExecutionResult, bool) {
	t.Helper()
	var result constants.ExecutionResult
	found := false
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		instruction, isHint, err := instructions.DetectInstruction(scanner.Bytes())
		if err != nil || !isHint || instruction == nil || instruction.Ref != ref || instruction.Name != constants.InstructionExecution {
			continue
		}
		raw, err := json.Marshal(instruction.Value)
		require.NoError(t, err)
		result = constants.ExecutionResult{}
		require.NoError(t, json.Unmarshal(raw, &result))
		found = true
	}
	return result, found
}
