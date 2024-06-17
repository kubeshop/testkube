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
