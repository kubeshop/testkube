package orchestration

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
)

var (
	Executions = newExecutionGroup(data.NewOutputProcessor(output.Std), output.Std)
)

type executionResult struct {
	ExitCode uint8
	Aborted  bool
}

type executionGroup struct {
	aborted   atomic.Bool
	outStream io.Writer
	errStream io.Writer

	executions   []*execution
	executionsMu sync.Mutex

	paused  atomic.Bool
	pauseMu sync.Mutex

	softKillProgress atomic.Bool
}

func newExecutionGroup(outStream io.Writer, errStream io.Writer) *executionGroup {
	return &executionGroup{
		outStream: outStream,
		errStream: errStream,
	}
}

func (e *executionGroup) Create(cmd string, args []string) *execution {
	// Instantiate the execution
	ex := &execution{group: e}
	ex.cmd = exec.Command(cmd, args...)
	ex.cmd.Stdout = e.outStream
	ex.cmd.Stderr = e.errStream

	// Append to the list TODO: delete that after finish
	e.executionsMu.Lock()
	e.executions = append(e.executions, ex)
	e.executionsMu.Unlock()

	return ex
}

func (e *executionGroup) CreateWithContext(ctx context.Context, cmd string, args []string) *execution {
	// Instantiate the execution
	ex := &execution{group: e}
	ex.cmd = exec.CommandContext(ctx, cmd, args...)
	ex.cmd.Stdout = e.outStream
	ex.cmd.Stderr = e.errStream

	// Append to the list TODO: delete that after finish
	e.executionsMu.Lock()
	e.executions = append(e.executions, ex)
	e.executionsMu.Unlock()

	return ex
}

func (e *executionGroup) Pause() (err error) {
	// Lock running
	swapped := e.paused.CompareAndSwap(false, true)
	if !swapped {
		return nil
	}
	e.pauseMu.Lock()

	// Lock the executions state
	e.executionsMu.Lock()
	defer e.executionsMu.Unlock()

	// Retrieve all started processes
	ps, totalFailure, err := processes()
	if totalFailure {
		return errors.Wrap(err, "failed to pause: failed to list processes")
	}
	if err != nil {
		output.Std.Direct().Warnf("warn: failed to pause: failed to list some processes: %v\n", err)
	}

	// Ignore the init process, to not suspend it accidentally
	ps.VirtualizePath(int32(os.Getpid()))
	err = ps.Suspend()
	return errors.Wrap(err, "failed to pause")
}

func (e *executionGroup) Resume() (err error) {
	// Lock running
	swapped := e.paused.CompareAndSwap(true, false)
	if !swapped {
		return nil
	}
	defer e.pauseMu.Unlock()

	// Lock the executions state
	e.executionsMu.Lock()
	defer e.executionsMu.Unlock()

	// Retrieve all started processes
	ps, totalFailure, err := processes()
	if totalFailure {
		return errors.Wrap(err, "failed to resume: failed to list processes")
	}
	if err != nil {
		output.Std.Direct().Warnf("warn: failed to resume: failed to list some processes: %v\n", err)
	}

	// Ignore the init process, to not suspend it accidentally
	ps.VirtualizePath(int32(os.Getpid()))
	err = ps.Resume()
	return errors.Wrap(err, "failed to resume")
}

func (e *executionGroup) Kill() (err error) {
	// Lock the executions state
	e.executionsMu.Lock()
	defer e.executionsMu.Unlock()

	// Skip if there are no executions to kill
	if len(e.executions) == 0 {
		return nil
	}

	// Retrieve all started processes
	ps, totalFailure, err := processes()
	if totalFailure {
		return errors.Wrap(err, "failed to kill: failed to list processes")
	}
	if err != nil {
		output.Std.Direct().Warnf("warn: failed to kill: failed to list some processes: %v\n", err.Error())
	}

	// Ignore the init process, to not suspend it accidentally
	ps.VirtualizePath(int32(os.Getpid()))
	err = ps.Kill()
	return errors.Wrap(err, "failed to kill")
}

func (e *executionGroup) Abort() {
	e.aborted.Store(true)
	_ = e.Kill()
	_ = e.Resume()
}

func (e *executionGroup) IsAborted() bool {
	return e.aborted.Load()
}

func (e *executionGroup) ClearAbortedStatus() {
	e.aborted.Store(false)
}

type execution struct {
	cmd   *exec.Cmd
	cmdMu sync.Mutex
	group *executionGroup
}

func (e *execution) Run() (*executionResult, error) {
	// Immediately fail when aborted
	if e.group.aborted.Load() {
		return &executionResult{Aborted: true, ExitCode: constants.CodeAborted}, nil
	}

	// Ensure it's not paused
	e.group.pauseMu.Lock()

	// Ensure the command is not running multiple times
	e.cmdMu.Lock()

	// Immediately fail when aborted
	if e.group.aborted.Load() {
		e.group.pauseMu.Unlock()
		e.cmdMu.Unlock()
		return &executionResult{Aborted: true, ExitCode: constants.CodeAborted}, nil
	}

	// Initialize local state
	var exitCode int
	var exitDetails string
	var aborted bool

	// Run the command
	err := e.cmd.Start()
	if err == nil {
		e.group.pauseMu.Unlock()
		e.cmdMu.Unlock()
		aborted, exitDetails, exitCode = getProcessStatus(e.cmd.Wait())
		if exitCode < 0 {
			exitCode = 255
			// Handle edge case, when i.e. EPIPE happened
			if !aborted {
				aborted = true
				fmt.Fprintf(e.cmd.Stderr, "\nThe process has been corrupted: %s.\n", exitDetails)
			}
		}
	} else {
		e.group.pauseMu.Unlock()
		e.cmdMu.Unlock()
		aborted, _, exitCode = getProcessStatus(err)
		if exitCode < 0 {
			exitCode = 255
		}
		e.cmd.Stderr.Write(append([]byte(err.Error()), '\n'))
	}

	// Clean up
	e.cmdMu.Lock()
	e.cmd = nil
	e.cmdMu.Unlock()

	// Mark the execution group as aborted when this process was aborted.
	// In Kubernetes, when that child process is killed, it may mean OOM Kill.
	if aborted && !e.group.aborted.Load() && !e.group.softKillProgress.Load() {
		e.group.Abort()
	}

	// Fail when aborted
	if e.group.aborted.Load() {
		return &executionResult{Aborted: true, ExitCode: constants.CodeAborted}, nil
	}

	return &executionResult{ExitCode: uint8(exitCode)}, nil
}

func getProcessStatus(err error) (bool, string, int) {
	if err == nil {
		return false, "", 0
	}
	if e, ok := err.(*exec.ExitError); ok {
		if e.ProcessState != nil {
			details := e.String()
			return details == "signal: killed", details, e.ExitCode()
		}
		return false, "", 1
	}
	return false, "", 1
}
