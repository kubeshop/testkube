// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package output

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
)

func Failf(exitCode uint8, message string, args ...interface{}) {
	// Print message
	fmt.Printf(message+"\n", args...)

	// Kill the sub-process
	if data.Step.Cmd != nil && data.Step.Cmd.Process != nil {
		_ = data.Step.Cmd.Process.Kill()
	}

	// Exit
	os.Exit(int(exitCode))
}
