package main

import (
	"errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	// Set verbosity
	ui.SetVerbose(env.Debug())

	// Validate provided data
	if env.Namespace() == "" || env.Ref() == "" {
		ui.Fail(errors.New("environment is misconfigured"))
	}

	commands.Execute()
}
