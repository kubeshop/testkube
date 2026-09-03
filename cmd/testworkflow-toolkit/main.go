package main

import (
	"errors"
	"os"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	// The local artifact receiver is run-owned plumbing, not a TestWorkflow
	// action. It deliberately has no TK_CFG or TK_REF because putting the relay
	// token or endpoint into that Pod annotation would disclose a credential.
	// Dispatch it before the ordinary action-runtime configuration guard.
	if isLocalArtifactReceiver(os.Args) {
		commands.Execute()
		return
	}

	// Set verbosity
	ui.SetVerbose(config.Debug())

	// Validate provided data
	if config.Namespace() == "" || config.Ref() == "" {
		ui.Fail(errors.New("environment is misconfigured"))
	}

	commands.Execute()
}

func isLocalArtifactReceiver(args []string) bool {
	return len(args) > 1 && args[1] == "local-artifact-receiver"
}
