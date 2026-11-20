package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testtriggers"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflowtemplates"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/webhooks"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/webhooktemplates"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/workflowtriggers"
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

func TestIsCompletionRequest(t *testing.T) {
	for _, args := range [][]string{
		{cobra.ShellCompRequestCmd, "get", ""},
		{cobra.ShellCompNoDescRequestCmd, "get", ""},
	} {
		t.Run(args[0], func(t *testing.T) {
			var seen bool
			root := &cobra.Command{
				Use:              "testkube",
				PersistentPreRun: func(cmd *cobra.Command, _ []string) { seen = isCompletionRequest(cmd) },
			}
			root.AddCommand(&cobra.Command{Use: "get", Run: func(*cobra.Command, []string) {}})
			root.SetArgs(args)
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})

			if err := root.Execute(); err != nil {
				t.Fatalf("executing %v: %v", args, err)
			}
			if !seen {
				t.Fatalf("expected %q to be treated as a completion request", args[0])
			}
		})
	}

	if isCompletionRequest(&cobra.Command{Use: "get"}) {
		t.Fatal("a regular command must not be treated as a completion request")
	}
}

func TestNameArgsHaveCompletion(t *testing.T) {
	for name, cmd := range map[string]*cobra.Command{
		"get testworkflow":              testworkflows.NewGetTestWorkflowsCmd(),
		"delete testworkflow":           testworkflows.NewDeleteTestWorkflowCmd(),
		"run testworkflow":              testworkflows.NewRunTestWorkflowCmd(),
		"abort testworkflowexecutions":  testworkflows.NewAbortTestWorkflowExecutionsCmd(),
		"cancel testworkflowexecutions": testworkflows.NewCancelTestWorkflowExecutionsCmd(),
		"get testworkflowtemplate":      testworkflowtemplates.NewGetTestWorkflowTemplatesCmd(),
		"delete testworkflowtemplate":   testworkflowtemplates.NewDeleteTestWorkflowTemplateCmd(),
		"get testworkflowexecution":     testworkflows.NewGetTestWorkflowExecutionsCmd(),
		"watch testworkflowexecution":   testworkflows.NewWatchTestWorkflowExecutionCmd(),
		"rerun testworkflowexecution":   testworkflows.NewReRunTestWorkflowExecutionCmd(),
		"cancel testworkflowexecution":  testworkflows.NewCancelTestWorkflowExecutionCmd(),
		"abort testworkflowexecution":   testworkflows.NewAbortTestWorkflowExecutionCmd(),
		"get testtrigger":               testtriggers.NewGetTestTriggerCmd(),
		"delete testtrigger":            testtriggers.NewDeleteTestTriggerCmd(),
		"get workflowtrigger":           workflowtriggers.NewGetWorkflowTriggerCmd(),
		"delete workflowtrigger":        workflowtriggers.NewDeleteWorkflowTriggerCmd(),
		"get webhook":                   webhooks.NewGetWebhookCmd(),
		"delete webhook":                webhooks.NewDeleteWebhookCmd(),
		"get webhooktemplate":           webhooktemplates.NewGetWebhookTemplateCmd(),
		"delete webhooktemplate":        webhooktemplates.NewDeleteWebhookTemplateCmd(),
	} {
		if cmd.ValidArgsFunction == nil {
			t.Errorf("%s: expected the name argument to have shell completion", name)
		}
	}

	cmd := testworkflows.NewGetTestWorkflowExecutionsCmd()
	if _, ok := cmd.GetFlagCompletionFunc("testworkflow"); !ok {
		t.Error("get testworkflowexecution: expected --testworkflow to have shell completion")
	}
}
