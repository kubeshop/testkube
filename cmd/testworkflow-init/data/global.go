package data

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/expressions/libs"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func GetBaseTestWorkflowMachine() expressions.Machine {
	var wd, err = os.Getwd()
	if err != nil {
		fmt.Printf("warn: problem reading working directory: %s\n", err.Error())
		wd = "/"
	}
	fileMachine := libs.NewFsMachine(os.DirFS("/"), wd)
	state := GetState() // load state
	executionCfg := state.InternalConfig.Execution
	executionMachine := testworkflowconfig.CreateExecutionMachine(&executionCfg)
	return expressions.CombinedMachines(EnvMachine, StateMachine, fileMachine, executionMachine)
}
