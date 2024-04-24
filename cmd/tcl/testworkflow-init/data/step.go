// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	gopsutil "github.com/shirou/gopsutil/v3/process"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/constants"
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
	out := NewOutputProcessor(s.Ref, os.Stdout)
	s.cmd.Stdout = out
	s.cmd.Stderr = os.Stderr
	s.cmd.Stdin = os.Stdin

	// Initialize local state
	var success bool
	var exitCode uint8

	// Run the command
	err := s.cmd.Start()
	if err == nil {
		s.pauseMu.Unlock()
		s.cmdMu.Unlock()
		success, exitCode = getProcessStatus(s.cmd.Wait())
	} else {
		s.pauseMu.Unlock()
		s.cmdMu.Unlock()
		success, exitCode = getProcessStatus(err)
	}

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
	s.cmdMu.Lock()
	if s.cmd != nil && s.cmd.Process != nil {
		ps, totalFailure, err2 := processes()
		if err2 != nil && totalFailure {
			err = err2
		} else {
			err = each(int32(s.cmd.Process.Pid), ps, func(p *gopsutil.Process) error {
				return p.Suspend()
			})
		}
	}
	s.cmdMu.Unlock()

	// Display output
	PrintHintDetails(s.Ref, "pause", t.Format(constants.PreciseTimeFormat))
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
	if s.cmd != nil && s.cmd.Process != nil {
		ps, totalFailure, err2 := processes()
		if err2 != nil && totalFailure {
			err = err2
		} else {
			err = each(int32(s.cmd.Process.Pid), ps, func(p *gopsutil.Process) error {
				return p.Resume()
			})
		}
	}
	s.cmdMu.Unlock()
	s.pauseMu.Unlock()

	// Display output
	PrintHintDetails(s.Ref, "resume", time.Now().Format(constants.PreciseTimeFormat))
	return err
}

func processes() (map[int32]int32, bool, error) {
	// Get list of processes
	list, err := gopsutil.Processes()
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to list processes")
	}
	ownPid := os.Getpid()

	// Get parent process for each process
	r := map[int32]int32{}
	var errs []error
	for _, p := range list {
		if p.Pid == int32(ownPid) {
			continue
		}
		r[p.Pid], err = p.Ppid()
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Return info
	if len(errs) > 0 {
		err = errors.Wrapf(errs[0], "failed to load %d/%d processes", len(errs), len(r))
	}
	return r, len(errs) == len(r), err
}

func each(pid int32, pidToPpid map[int32]int32, fn func(*gopsutil.Process) error) error {
	if _, ok := pidToPpid[pid]; !ok {
		return fmt.Errorf("process %d: not found", pid)
	}

	// Run operation for the process
	err := fn(&gopsutil.Process{Pid: pid})
	if err != nil {
		return errors.Wrapf(err, "process %d: failed to perform", pid)
	}

	// Run operation for all the children recursively
	for p, ppid := range pidToPpid {
		if ppid == pid {
			err = each(p, pidToPpid, fn)
			if err != nil {
				return errors.Wrapf(err, "process %d: children", pid)
			}
		}
	}

	return nil
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
