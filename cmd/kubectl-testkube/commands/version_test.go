package commands

import (
	"testing"
)

// TestVersionCmdShape is a smoke test ensuring NewVersionCmd returns a
// well-formed cobra command with the expected metadata. The bulk of the
// update-awareness behavior is exercised by the common package tests for
// CheckComponentsStatus and MaybeNotifyNewerRelease; this test just guards
// against accidental regressions in the command wiring.
func TestVersionCmdShape(t *testing.T) {
	cmd := NewVersionCmd()
	if cmd == nil {
		t.Fatal("NewVersionCmd returned nil")
	}
	if cmd.Use != "version" {
		t.Fatalf("expected Use=version, got %q", cmd.Use)
	}
	if cmd.Run == nil {
		t.Fatal("expected version command to define a Run function")
	}
	if cmd.PersistentPreRun == nil {
		t.Fatal("expected version command to define a PersistentPreRun function")
	}

	foundAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "v" {
			foundAlias = true
			break
		}
	}
	if !foundAlias {
		t.Fatalf("expected 'v' alias on version command, got %v", cmd.Aliases)
	}
}
