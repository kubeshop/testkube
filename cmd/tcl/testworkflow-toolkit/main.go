// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package main

import (
	"errors"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
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
