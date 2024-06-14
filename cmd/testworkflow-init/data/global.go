// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import (
	"os"

	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/expressions/libs"
)

func GetBaseTestWorkflowMachine() expressions.Machine {
	var wd, _ = os.Getwd()
	fileMachine := libs.NewFsMachine(os.DirFS("/"), wd)
	LoadState()
	return expressions.CombinedMachines(EnvMachine, StateMachine, fileMachine)
}
