package ui

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestPromptsRefuseNonInteractiveStdin runs the guard in a subprocess because it exits the
// process on purpose. Without the guard these prompts spin forever on a non-terminal stdin
// instead of returning, so the test would hang rather than fail.
func TestPromptsRefuseNonInteractiveStdin(t *testing.T) {
	t.Parallel()

	for _, prompt := range []string{"select", "confirm", "textinput"} {
		t.Run(prompt, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, testBinary(t), "-test.run=TestPromptHelperUnderGuard")
			cmd.Env = append(cmd.Environ(), "TESTKUBE_UI_PROMPT="+prompt)
			out, err := cmd.CombinedOutput()

			if err == nil {
				t.Fatalf("%s prompt was answered without a terminal, output: %s", prompt, out)
			}
			if !strings.Contains(string(out), "stdin is not a terminal") {
				t.Fatalf("expected the guard to say why it refused, got: %s", out)
			}
		})
	}
}

func TestPromptHelperUnderGuard(t *testing.T) {
	prompt := promptUnderTest()
	if prompt == "" {
		t.Skip("only runs as a subprocess of TestPromptsRefuseNonInteractiveStdin")
	}

	stdinIsInteractive = func() bool { return false }

	u := NewStdoutUI(false)
	switch prompt {
	case "select":
		u.Select("pick one", []string{"a", "b"})
	case "confirm":
		u.Confirm("are you sure")
	case "textinput":
		u.TextInput("your name")
	}

	t.Fatalf("%s returned instead of refusing", prompt)
}
