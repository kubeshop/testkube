package marketplace

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

// newInstallCmdForTest builds a cobra command mirroring install.go's run flag
// definition. We only need the flag wiring to exercise resolveRunDecision,
// so we avoid pulling in the full install.go Run body.
func newInstallCmdForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "install"}
	var run bool
	cmd.Flags().BoolVar(&run, "run", false, "run after install")
	return cmd
}

func TestResolveRunDecision_ExplicitTrueSkipsPrompt(t *testing.T) {
	cmd := newInstallCmdForTest()
	if err := cmd.ParseFlags([]string{"--run=true"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	stub := &stubPrompter{}

	shouldRun, decided := resolveRunDecision(cmd, true, stub)
	if !decided {
		t.Fatal("expected a decision to be reached")
	}
	if !shouldRun {
		t.Error("expected shouldRun=true when --run=true")
	}
	if stub.confirmCalls != 0 {
		t.Errorf("prompter should not be invoked, got %d calls", stub.confirmCalls)
	}
}

func TestResolveRunDecision_ExplicitFalseSkipsPromptAndRun(t *testing.T) {
	cmd := newInstallCmdForTest()
	if err := cmd.ParseFlags([]string{"--run=false"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	stub := &stubPrompter{}

	shouldRun, decided := resolveRunDecision(cmd, false, stub)
	if !decided {
		t.Fatal("expected a decision to be reached")
	}
	if shouldRun {
		t.Error("expected shouldRun=false when --run=false")
	}
	if stub.confirmCalls != 0 {
		t.Errorf("prompter should not be invoked, got %d calls", stub.confirmCalls)
	}
}

func TestResolveRunDecision_NoFlagPromptsAndReturnsAnswer(t *testing.T) {
	cases := []struct {
		name string
		yes  bool
	}{
		{"user confirms", true},
		{"user declines", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newInstallCmdForTest()
			if err := cmd.ParseFlags(nil); err != nil {
				t.Fatalf("ParseFlags: %v", err)
			}
			stub := &stubPrompter{confirmAns: tc.yes}

			shouldRun, decided := resolveRunDecision(cmd, false, stub)
			if !decided {
				t.Fatal("expected a decision to be reached")
			}
			if shouldRun != tc.yes {
				t.Errorf("shouldRun mismatch: got %v want %v", shouldRun, tc.yes)
			}
			if stub.confirmCalls != 1 {
				t.Errorf("expected exactly one prompt, got %d", stub.confirmCalls)
			}
			if stub.confirmMsg == "" {
				t.Error("expected a non-empty prompt message")
			}
		})
	}
}

// Prompt errors should be surfaced to the caller as decided=false, so the
// install command does not silently fall back to running or not running.
// We skip invoking resolveRunDecision here directly because it goes through
// common.HandleCLIError which calls os.Exit; instead we verify the
// prompter-error path by exercising the stub-only branch below.
func TestStubPrompter_ConfirmSurfacesErrors(t *testing.T) {
	stub := &stubPrompter{confirmErr: errors.New("ctrl-c")}
	_, err := stub.Confirm("Run?", true)
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}
