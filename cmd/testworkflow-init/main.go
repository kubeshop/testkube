package main

import (
	"os"
	"strconv"

	"github.com/gookit/color"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runner"
)

func main() {
	// Force colors
	color.ForceColor()

	// Initialize orchestration setup early
	orchestration.Initialize()

	// Ensure there is a group index provided
	if len(os.Args) != 2 {
		output.ExitErrorf(constants.CodeInternal, "invalid arguments provided - expected only one")
	}

	// Determine group index to run
	groupIndex, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		output.ExitErrorf(constants.CodeInputError, "invalid run group passed: %s", err.Error())
	}

	// Run the init process
	exitCode, err := runner.RunInit(int(groupIndex))
	if err != nil {
		output.ExitErrorf(uint8(exitCode), "%s", err.Error())
	}
	os.Exit(exitCode)
}
