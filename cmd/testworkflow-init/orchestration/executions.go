package orchestration

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

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
}

func (e *executionGroup) IsAborted() bool {
	return e.aborted.Load()
}

type execution struct {
	cmd   *exec.Cmd
	cmdMu sync.Mutex
	group *executionGroup
}

func (e *execution) Run() (*executionResult, error) {
	// Immediately fail when aborted
	if e.group.aborted.Load() {
		return &executionResult{Aborted: true, ExitCode: data.CodeAborted}, nil
	}

	// Ensure it's not paused
	e.group.pauseMu.Lock()

	// Ensure the command is not running multiple times
	e.cmdMu.Lock()

	// Immediately fail when aborted
	if e.group.aborted.Load() {
		e.group.pauseMu.Unlock()
		e.cmdMu.Unlock()
		return &executionResult{Aborted: true, ExitCode: data.CodeAborted}, nil
	}

	// Initialize local state
	var exitCode uint8

	// Run the command
	err := e.cmd.Start()
	if err == nil {
		e.group.pauseMu.Unlock()
		e.cmdMu.Unlock()
		_, exitCode = getProcessStatus(e.cmd.Wait())
	} else {
		e.group.pauseMu.Unlock()
		e.cmdMu.Unlock()
		_, exitCode = getProcessStatus(err)
		e.cmd.Stderr.Write(append([]byte(err.Error()), '\n'))
	}

	// Clean up
	e.cmdMu.Lock()
	e.cmd = nil
	e.cmdMu.Unlock()

	// Fail when aborted
	if e.group.aborted.Load() {
		return &executionResult{Aborted: true, ExitCode: data.CodeAborted}, nil
	}

	return &executionResult{ExitCode: exitCode}, nil
}

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
	return false, 1
}
