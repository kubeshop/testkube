package data

import (
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
)

var Step = &step{}

type step struct {
	Ref        string
	Status     StepStatus
	ExitCode   uint8
	Executed   bool
	InitStatus string

	paused      atomic.Bool
	pausedNs    atomic.Int64
	pausedStart time.Time
	cmd         *exec.Cmd
	runMu       sync.Mutex
	cmdMu       sync.Mutex
	pauseMu     sync.Mutex
}

// TODO: Obfuscate Stdout/Stderr streams
func (s *step) Run(negative bool, cmd string, args ...string) {
	// Avoid multiple runs at once
	s.runMu.Lock()
	defer s.runMu.Unlock()

	// Wait until not paused
	s.pauseMu.Lock()

	// Prepare the command
	s.cmdMu.Lock()
	s.cmd = exec.Command(cmd, args...)
	out := NewOutputProcessor(os.Stdout)
	s.cmd.Stdout = out
	s.cmd.Stderr = os.Stderr
	s.cmd.Stdin = os.Stdin

	// Initialize local state
	var success bool
	var exitCode uint8

	// Run the command
	//err := s.cmd.Start()
	//if err == nil {
	//	s.pauseMu.Unlock()
	//	s.cmdMu.Unlock()
	//	success, exitCode = getProcessStatus(s.cmd.Wait())
	//} else {
	//	s.pauseMu.Unlock()
	//	s.cmdMu.Unlock()
	//	success, exitCode = getProcessStatus(err)
	//}

	s.ExitCode = exitCode
	if negative {
		success = !success
	}
	if success {
		s.Status = StepStatusPassed
	} else {
		s.Status = StepStatusFailed
	}

	// Clean up
	s.cmdMu.Lock()
	s.cmd = nil
	s.cmdMu.Unlock()
}

func (s *step) Took(since time.Time) time.Duration {
	now := time.Now()
	if s.paused.Load() {
		now = s.pausedStart
	}
	if !now.After(since) {
		return 0
	}
	return now.Sub(since) - time.Duration(s.pausedNs.Load())
}

func (s *step) Kill() {
	s.cmdMu.Lock()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	s.cmdMu.Unlock()
}

func (s *step) Pause(t time.Time) (err error) {
	// Lock running
	swapped := s.paused.CompareAndSwap(false, true)
	if !swapped {
		return nil
	}
	s.pauseMu.Lock()

	// Save the information about current pause time
	s.pausedStart = time.Now()

	// Pause already started application
	//s.cmdMu.Lock()
	//if s.cmd != nil && s.cmd.Process != nil {
	//	ps, totalFailure, err2 := processes()
	//	if err2 != nil && totalFailure {
	//		err = err2
	//	} else {
	//		err = each(int32(s.cmd.Process.Pid), ps, func(p *gopsutil.Process) error {
	//			return p.Suspend()
	//		})
	//	}
	//}
	//s.cmdMu.Unlock()

	// Display output
	PrintHintDetails(s.Ref, constants.InstructionPause, t.Format(constants.PreciseTimeFormat))
	return err
}

func (s *step) Resume() (err error) {
	// Unlock running
	swapped := s.paused.CompareAndSwap(true, false)
	if !swapped {
		return nil
	}

	// Finish current pause period
	s.pausedNs.Add(time.Now().Sub(s.pausedStart).Nanoseconds())

	// Resume started application
	s.cmdMu.Lock()
	//if s.cmd != nil && s.cmd.Process != nil {
	//	ps, totalFailure, err2 := processes()
	//	if err2 != nil && totalFailure {
	//		err = err2
	//	} else {
	//		err = each(int32(s.cmd.Process.Pid), ps, func(p *gopsutil.Process) error {
	//			return p.Resume()
	//		})
	//	}
	//}
	s.cmdMu.Unlock()
	s.pauseMu.Unlock()

	// Display output
	PrintHintDetails(s.Ref, constants.InstructionResume, time.Now().Format(constants.PreciseTimeFormat))
	return err
}
