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

	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl/libs"
)

func GetBaseTestWorkflowMachine() expressionstcl.Machine {
	var wd, _ = os.Getwd()
	fileMachine := libs.NewFsMachine(os.DirFS("/"), wd)
	LoadState()
	return expressionstcl.CombinedMachines(EnvMachine, StateMachine, fileMachine)
}
