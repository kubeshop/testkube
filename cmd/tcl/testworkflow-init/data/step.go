// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import "C"
import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	gopsutil "github.com/shirou/gopsutil/v3/process"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/utils"
)

var Step = &step{}

type step struct {
	Ref        string
	Status     StepStatus
	ExitCode   uint8
	Executed   bool
	InitStatus string

	paused  atomic.Bool
	cmd     *exec.Cmd
	runMu   sync.Mutex
	cmdMu   sync.Mutex
	pauseMu sync.Mutex
}

// TODO: Obfuscate Stdout/Stderr streams
func (s *step) Run(negative bool, cmd string, args ...string) {
	// Avoid multiple runs at once
	s.runMu.Lock()
	defer s.runMu.Unlock()

	// Wait until not paused
	s.pauseMu.Lock()
	s.pauseMu.Unlock()

	// Prepare the command
	s.cmdMu.Lock()
	s.cmd = exec.Command(cmd, args...)
	out := utils.NewOutputProcessor(s.Ref, os.Stdout)
	s.cmd.Stdout = out
	s.cmd.Stderr = os.Stderr
	s.cmd.Stdin = os.Stdin

	// Initialize local state
	var success bool
	var exitCode uint8

	// Run the command
	err := s.cmd.Start()
	if err == nil {
		s.cmdMu.Unlock()
		success, exitCode = getProcessStatus(s.cmd.Wait())
	} else {
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

func (s *step) Pause() {
	// Lock running
	swapped := s.paused.CompareAndSwap(false, true)
	if !swapped {
		return
	}
	s.pauseMu.Lock()

	// TODO: Save the information about current pause time

	// Pause already started application
	s.cmdMu.Lock()
	if s.cmd != nil && s.cmd.Process != nil {
		err := each(int32(s.cmd.Process.Pid), func(p *gopsutil.Process) error {
			return p.Suspend()
		})
		if err != nil {
			fmt.Printf("Warning: failed to pause: %s\n", err)
		}
	}
	s.cmdMu.Unlock()

	// Display output
	PrintOutput(s.Ref, "pause", time.Now())
}

func (s *step) Resume() {
	// Unlock running
	swapped := s.paused.CompareAndSwap(true, false)
	if !swapped {
		return
	}

	// TODO: Finish current pause period

	// Resume started application
	s.cmdMu.Lock()
	if s.cmd != nil && s.cmd.Process != nil {
		err := each(int32(s.cmd.Process.Pid), func(p *gopsutil.Process) error {
			return p.Resume()
		})
		if err != nil {
			fmt.Printf("Warning: failed to resume: %s\n", err)
		}
	}
	s.cmdMu.Unlock()
	s.pauseMu.Unlock()

	// Display output
	PrintOutput(s.Ref, "resume", time.Now())
}

func each(pid int32, fn func(*gopsutil.Process) error) error {
	p := &gopsutil.Process{Pid: pid}
	err := fn(p)
	if err != nil {
		return errors.Wrapf(err, "process %d: failed to perform", pid)
	}
	children, err := p.Children()
	if err != nil {
		return errors.Wrapf(err, "process %d: failed to get children", pid)
	}
	for _, child := range children {
		err := each(child.Pid, fn)
		if err != nil {
			return errors.Wrapf(err, "process %d", pid)
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
