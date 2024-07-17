package orchestration

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
)

var (
	Executions = newExecutionGroup(data.NewOutputProcessor(os.Stdout), os.Stderr)
)

type executionResult struct {
	ExitCode uint8
}

type executionGroup struct {
	aborted   atomic.Bool
	outStream io.Writer
	errStream io.Writer

	executions   []*execution
	executionsMu sync.Mutex

	paused      atomic.Bool
	pausedNs    atomic.Int64
	pausedStart time.Time
	pauseMu     sync.Mutex
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

func (e *executionGroup) PauseAll() (err error) {
	// Lock running
	swapped := e.paused.CompareAndSwap(false, true)
	if !swapped {
		return nil
	}
	e.pauseMu.Lock()

	// Save the information about current pause time
	e.pausedStart = time.Now()

	// Lock the executions state
	e.executionsMu.Lock()
	defer e.executionsMu.Unlock()

	// Retrieve all started processes
	ps, totalFailure, err := processes()
	if totalFailure {
		return errors.Wrap(err, "failed to pause: failed to list processes")
	}
	if err != nil {
		fmt.Printf("warning: failed to pause: failed to list some processes: %v\n", err.Error())
	}

	// Ignore the init process, to not suspend it accidentally
	ps.VirtualizePath(int32(os.Getpid()))
	err = ps.Suspend()
	return errors.Wrap(err, "failed to pause")

	// Display output TODO
	//PrintHintDetails(s.Ref, constants.InstructionPause, t.Format(constants.PreciseTimeFormat))
}

func (e *executionGroup) ResumeAll() (err error) {
	// Lock running
	swapped := e.paused.CompareAndSwap(true, false)
	if !swapped {
		return nil
	}
	defer e.pauseMu.Unlock()

	// Finish current pause period TODO: is it needed?
	e.pausedNs.Add(time.Now().Sub(e.pausedStart).Nanoseconds())

	// Lock the executions state
	e.executionsMu.Lock()
	defer e.executionsMu.Unlock()

	// Retrieve all started processes
	ps, totalFailure, err := processes()
	if totalFailure {
		return errors.Wrap(err, "failed to resume: failed to list processes")
	}
	if err != nil {
		fmt.Printf("warning: failed to resume: failed to list some processes: %v\n", err.Error())
	}

	// Ignore the init process, to not suspend it accidentally
	ps.VirtualizePath(int32(os.Getpid()))
	err = ps.Resume()
	return errors.Wrap(err, "failed to resume")

	// Display output TODO
	//PrintHintDetails(s.Ref, constants.InstructionResume, t.Format(constants.PreciseTimeFormat))
}

func (e *executionGroup) KillAll() (err error) {
	// Lock the executions state
	e.executionsMu.Lock()
	defer e.executionsMu.Unlock()

	// Retrieve all started processes
	ps, totalFailure, err := processes()
	if totalFailure {
		return errors.Wrap(err, "failed to resume: failed to list processes")
	}
	if err != nil {
		fmt.Printf("warning: failed to resume: failed to list some processes: %v\n", err.Error())
	}

	// Ignore the init process, to not suspend it accidentally
	ps.VirtualizePath(int32(os.Getpid()))
	err = ps.Kill()
	return errors.Wrap(err, "failed to resume")

	// Display output TODO
	//PrintHintDetails(s.Ref, constants.InstructionResume, t.Format(constants.PreciseTimeFormat))
}

func (e *executionGroup) Abort() {
	e.aborted.Store(true)
	_ = e.KillAll()
}

func (e *executionGroup) IsAborted() bool {
	return e.aborted.Load()
}

type execution struct {
	cmd   *exec.Cmd
	runMu sync.Mutex
	cmdMu sync.Mutex
	group *executionGroup
}

func (e *execution) Run() (*executionResult, error) {
	// Immediately fail when aborted
	if e.group.aborted.Load() {
		return &executionResult{ExitCode: data.CodeAborted}, nil
	}

	// Ensure it's not paused
	e.group.pauseMu.Lock()

	// Ensure the command is not running multiple times
	e.cmdMu.Lock()

	// Immediately fail when aborted
	if e.group.aborted.Load() {
		e.group.pauseMu.Unlock()
		e.cmdMu.Unlock()
		return &executionResult{ExitCode: data.CodeAborted}, nil
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
	}

	// Clean up
	e.cmdMu.Lock()
	e.cmd = nil
	e.cmdMu.Unlock()

	// Fail when aborted
	if e.group.aborted.Load() {
		exitCode = data.CodeAborted
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
	fmt.Println(err.Error())
	return false, 1
}
