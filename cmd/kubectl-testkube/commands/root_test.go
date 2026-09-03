package commands

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
)

func TestIsLocalCommandChecksFullAncestry(t *testing.T) {
	root := &cobra.Command{Use: "testkube"}
	local := &cobra.Command{Use: "local"}
	run := &cobra.Command{Use: "run"}
	clean := &cobra.Command{Use: "clean"}
	root.AddCommand(local)
	local.AddCommand(run, clean)

	for _, cmd := range []*cobra.Command{local, run, clean} {
		if !isLocalCommand(cmd) {
			t.Fatalf("expected %q to be recognized as a local command", cmd.CommandPath())
		}
	}

	if isLocalCommand(root) {
		t.Fatal("root command must not be recognized as local")
	}

	get := &cobra.Command{Use: "get"}
	root.AddCommand(get)
	if isLocalCommand(get) {
		t.Fatal("unrelated command must not be recognized as local")
	}
}

func TestRootLifecycleHooksBypassLocalCommand(t *testing.T) {
	root := &cobra.Command{Use: "testkube"}
	local := &cobra.Command{Use: "local"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(local)
	local.AddCommand(run)

	if RootCmd.PersistentPreRun == nil || RootCmd.PersistentPostRun == nil {
		t.Fatal("expected root persistent lifecycle hooks")
	}

	// Both hooks return before config loading, Cloud-context validation, API
	// client construction, telemetry, and update checks for a local descendant.
	// Calling them directly keeps this unit test independent of the user's
	// Testkube configuration and demonstrates the offline guard at the hook.
	RootCmd.PersistentPreRun(run, nil)
	RootCmd.PersistentPostRun(run, nil)
}

type testExitCodeError struct {
	code int
}

func (e testExitCodeError) Error() string {
	return "typed command error"
}

func (e testExitCodeError) ExitCode() int {
	return e.code
}

func TestExitCodeForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "ordinary errors retain historical exit status",
			err:  fmt.Errorf("ordinary command error"),
			want: 1,
		},
		{
			name: "typed error preserves usage exit status",
			err:  testExitCodeError{code: 2},
			want: 2,
		},
		{
			name: "wrapped typed error preserves cancellation exit status",
			err:  fmt.Errorf("running local command: %w", testExitCodeError{code: 130}),
			want: 130,
		},
		{
			name: "invalid zero typed status falls back to failure",
			err:  testExitCodeError{code: 0},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCodeForError(tt.err); got != tt.want {
				t.Fatalf("exitCodeForError() = %d, want %d", got, tt.want)
			}
		})
	}
}
