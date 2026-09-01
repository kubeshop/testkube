package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestZshCompletionUsesActualBinaryName guards against #964: the zsh
// completion script must reference the binary's actual invoked name
// ("kubectl-testkube"), not RootCmd.Use ("testkube"), or the installed
// script never matches and zsh completion silently does nothing.
func TestZshCompletionUsesActualBinaryName(t *testing.T) {
	root := &cobra.Command{Use: "testkube"}
	root.AddCommand(&cobra.Command{Use: "get"})

	var buf bytes.Buffer
	if err := genZshCompletionAs(root, pluginBinaryName, &buf); err != nil {
		t.Fatalf("genZshCompletionAs returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "#compdef kubectl-testkube") {
		t.Fatalf("expected zsh script to declare #compdef kubectl-testkube, got:\n%s", out[:200])
	}
	if !strings.Contains(out, "_kubectl-testkube()") {
		t.Fatalf("expected zsh script to define _kubectl-testkube(), got:\n%s", out[:400])
	}
	if strings.Contains(out, "_testkube()") {
		t.Fatalf("zsh script should not define _testkube() when generated for kubectl-testkube")
	}

	// genZshCompletionAs must not leave Use mutated behind it.
	if root.Use != "testkube" {
		t.Fatalf("expected root.Use to be restored to %q, got %q", "testkube", root.Use)
	}
}

func TestCompletionCmdRejectsUnknownShell(t *testing.T) {
	cmd := NewCompletionCmd()
	cmd.SetArgs([]string{"invalidshell"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error for an unsupported shell argument")
	}
}
