package main

import (
	"errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	// Set verbosity
	ui.SetVerbose(config.Debug())
	ui.Info("Starting testworkflow-toolkit WITO HAS GONE AWAY")

	// Validate provided data
	if config.Namespace() == "" || config.Ref() == "" {
		ui.Fail(errors.New("environment is misconfigured"))
	}

	commands.Execute()
}
