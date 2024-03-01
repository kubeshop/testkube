// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package run

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
)

func getProcessStatus(err error) (bool, uint8) {
	if err == nil {
		return true, 0
	}
	if e, ok := err.(*exec.ExitError); ok {
		if e.ProcessState != nil {
			return false, uint8(e.ProcessState.ExitCode())
		}
		return false, 1
	}
	fmt.Println(err.Error())
	return false, 1
}

// TODO: Obfuscate Stdout/Stderr streams
func createCommand(cmd string, args ...string) (c *exec.Cmd) {
	c = exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return
}

func execute(cmd string, args ...string) {
	data.Step.Cmd = createCommand(cmd, args...)
	success, exitCode := getProcessStatus(data.Step.Cmd.Run())
	data.Step.ExitCode = exitCode

	actualSuccess := success
	if data.Config.Negative {
		actualSuccess = !success
	}

	if actualSuccess {
		data.Step.Status = data.StepStatusPassed
	} else {
		data.Step.Status = data.StepStatusFailed
	}

	if data.Config.Negative {
		fmt.Printf("Expected to fail: finished with exit code %d.\n", exitCode)
	} else if data.Config.Debug {
		fmt.Printf("Exit code: %d.\n", exitCode)
	}
}

func Run(cmd string, args []string) {
	// Instantiate the command and run
	execute(cmd, args...)

	// Retry if it's expected
	// TODO: Support nested retries
	step := data.State.GetStep(data.Step.Ref)
	for step.Iteration <= uint64(data.Config.RetryCount) {
		expr, err := data.Expression(data.Config.RetryUntil, data.LocalMachine)
		if err != nil {
			fmt.Printf("Failed to execute retry condition: %s: %s\n", data.Config.RetryUntil, err.Error())
			data.Finish()
		}
		v, _ := expr.BoolValue()
		if v {
			break
		}
		step.Next()
		fmt.Printf("\nExit code: %d • Retrying: attempt #%d (of %d):\n", data.Step.ExitCode, step.Iteration-1, data.Config.RetryCount)
		execute(cmd, args...)
	}

	// Finish
	data.Finish()
}
