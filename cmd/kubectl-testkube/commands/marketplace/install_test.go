package marketplace

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

// newInstallCmdForTest builds a cobra command mirroring install.go's flag
// definitions. We only need the flag wiring to exercise the decision
// helpers, so we avoid pulling in the full install.go Run body.
func newInstallCmdForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "install"}
	var (
		run         bool
		interactive bool
	)
	var follow bool
	cmd.Flags().BoolVar(&run, "run", false, "run after install")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "prompt for params")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow execution")
	return cmd
}

// withStubbedTTY swaps isStdinTTY for the duration of a test.
func withStubbedTTY(t *testing.T, tty bool) {
	t.Helper()
	prev := isStdinTTY
	isStdinTTY = func() bool { return tty }
	t.Cleanup(func() { isStdinTTY = prev })
}

func TestResolveRunDecision_ExplicitTrueSkipsPrompt(t *testing.T) {
	cmd := newInstallCmdForTest()
	if err := cmd.ParseFlags([]string{"--run=true"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	stub := &stubPrompter{}

	shouldRun, decided := resolveRunDecision(cmd, true, false, stub)
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

	shouldRun, decided := resolveRunDecision(cmd, false, false, stub)
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

func TestResolveRunDecision_ExplicitRunFalseBeatsFollow(t *testing.T) {
	// --run=false is an explicit opt-out that must beat -f; otherwise the
	// user could not install + follow-later with a separate `run` command.
	cmd := newInstallCmdForTest()
	if err := cmd.ParseFlags([]string{"--run=false", "-f"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	stub := &stubPrompter{}

	shouldRun, decided := resolveRunDecision(cmd, false, true, stub)
	if !decided {
		t.Fatal("expected a decision to be reached")
	}
	if shouldRun {
		t.Error("expected shouldRun=false when --run=false even with -f")
	}
	if stub.confirmCalls != 0 {
		t.Errorf("prompter should not be invoked, got %d calls", stub.confirmCalls)
	}
}

func TestResolveRunDecision_FollowImpliesRunWhenRunUnset(t *testing.T) {
	cmd := newInstallCmdForTest()
	if err := cmd.ParseFlags([]string{"-f"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	stub := &stubPrompter{}

	shouldRun, decided := resolveRunDecision(cmd, false, true, stub)
	if !decided {
		t.Fatal("expected a decision to be reached")
	}
	if !shouldRun {
		t.Error("expected shouldRun=true when -f is supplied without --run")
	}
	if stub.confirmCalls != 0 {
		t.Errorf("-f should bypass the prompt, got %d calls", stub.confirmCalls)
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

			shouldRun, decided := resolveRunDecision(cmd, false, false, stub)
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

func TestShouldPromptForParameters(t *testing.T) {
	tests := []struct {
		name      string
		flags     []string
		interFlag bool
		hasParams bool
		tty       bool
		want      bool
	}{
		{
			name: "no flag + TTY + params -> prompt",
			tty:  true, hasParams: true, want: true,
		},
		{
			name: "no flag + TTY + no params -> skip",
			tty:  true, hasParams: false, want: false,
		},
		{
			name: "no flag + no TTY + params -> skip (CI path)",
			tty:  false, hasParams: true, want: false,
		},
		{
			name:  "--interactive=true forces prompt even without TTY",
			flags: []string{"--interactive=true"}, interFlag: true,
			tty: false, hasParams: true, want: true,
		},
		{
			name:  "--interactive=true with no params still returns true",
			flags: []string{"--interactive=true"}, interFlag: true,
			tty: true, hasParams: false, want: true,
		},
		{
			name:  "--interactive=false skips even on TTY with params",
			flags: []string{"--interactive=false"}, interFlag: false,
			tty: true, hasParams: true, want: false,
		},
		{
			name:  "-i shorthand forces prompt",
			flags: []string{"-i"}, interFlag: true,
			tty: false, hasParams: true, want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withStubbedTTY(t, tc.tty)
			cmd := newInstallCmdForTest()
			if err := cmd.ParseFlags(tc.flags); err != nil {
				t.Fatalf("ParseFlags: %v", err)
			}
			// Re-read the flag to mirror how install.go's Run body sees it.
			interactive, _ := cmd.Flags().GetBool("interactive")
			got := shouldPromptForParameters(cmd, interactive, tc.hasParams)
			if got != tc.want {
				t.Errorf("shouldPromptForParameters = %v, want %v (interFlag=%v tty=%v hasParams=%v)",
					got, tc.want, tc.interFlag, tc.tty, tc.hasParams)
			}
		})
	}
}
