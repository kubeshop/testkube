// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowcontroller

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

type Instruction struct {
	Ref   string
	Name  string
	Value interface{}
}

func (i *Instruction) ToInternal() *testkube.TestWorkflowOutput {
	if i == nil {
		return nil
	}
	value := map[string]interface{}(nil)
	if i.Value != nil {
		v, _ := json.Marshal(i.Value)
		e := json.Unmarshal(v, &value)
		if e != nil {
			log.DefaultLogger.Warnf("invalid output passed from TestWorfklow - %v", i.Value)
		}
	}
	if v, ok := i.Value.(map[string]interface{}); ok {
		value = v
	}
	return &testkube.TestWorkflowOutput{
		Ref:   i.Ref,
		Name:  i.Name,
		Value: value,
	}
}

type Comment struct {
	Time   time.Time
	Hint   *Instruction
	Output *Instruction
}

type ContainerLog struct {
	Time   time.Time
	Log    []byte
	Hint   *Instruction
	Output *Instruction
}

type ContainerResult struct {
	Status     testkube.TestWorkflowStepStatus
	Details    string
	ExitCode   int
	FinishedAt time.Time
}

var UnknownContainerResult = ContainerResult{
	Status:   testkube.ABORTED_TestWorkflowStepStatus,
	ExitCode: -1,
}

func GetContainerResult(c corev1.ContainerStatus) ContainerResult {
	if c.State.Waiting != nil {
		return ContainerResult{Status: testkube.QUEUED_TestWorkflowStepStatus, ExitCode: -1}
	}
	if c.State.Running != nil {
		return ContainerResult{Status: testkube.RUNNING_TestWorkflowStepStatus, ExitCode: -1}
	}
	re := regexp.MustCompile(`^([^,]*),(0|[1-9]\d*)$`)

	// Workaround - GKE sends SIGKILL after the container is already terminated,
	// and the pod gets stuck then.
	if c.State.Terminated.Reason != "Completed" {
		return ContainerResult{Status: testkube.ABORTED_TestWorkflowStepStatus, Details: c.State.Terminated.Reason, ExitCode: -1, FinishedAt: c.State.Terminated.FinishedAt.Time}
	}

	msg := c.State.Terminated.Message
	match := re.FindStringSubmatch(msg)
	if match == nil {
		return ContainerResult{Status: testkube.ABORTED_TestWorkflowStepStatus, ExitCode: -1, FinishedAt: c.State.Terminated.FinishedAt.Time}
	}
	status := testkube.TestWorkflowStepStatus(match[1])
	exitCode, _ := strconv.Atoi(match[2])
	if status == "" {
		status = testkube.PASSED_TestWorkflowStepStatus
	}
	return ContainerResult{Status: status, ExitCode: exitCode, FinishedAt: c.State.Terminated.FinishedAt.Time}
}

func GetFinalContainerResult(ctx context.Context, pod Watcher[*corev1.Pod], containerName string) (ContainerResult, error) {
	w := WatchContainerStatus(ctx, pod, containerName, 0)
	stream := w.Stream(ctx)
	defer w.Close()

	for c := range stream.Channel() {
		if c.Error != nil {
			return UnknownContainerResult, c.Error
		}
		if c.Value.State.Terminated == nil {
			continue
		}
		return GetContainerResult(c.Value), nil
	}
	return UnknownContainerResult, nil
}

var ErrNoStartedEvent = errors.New("started event not received")

func WatchContainerPreEvents(ctx context.Context, podEvents Watcher[*corev1.Event], containerName string, cacheSize int, includePodWarnings bool) Watcher[*corev1.Event] {
	w := newWatcher[*corev1.Event](ctx, cacheSize)
	go func() {
		events := WatchContainerEvents(ctx, podEvents, containerName, 0, includePodWarnings)
		defer events.Close()
		defer w.Close()

		for ev := range events.Stream(ctx).Channel() {
			if ev.Error != nil {
				w.SendError(ev.Error)
			} else {
				w.SendValue(ev.Value)
				if ev.Value.Reason == "Started" {
					return
				}
			}
		}
	}()
	return w
}

func WatchPodPreEvents(ctx context.Context, podEvents Watcher[*corev1.Event], cacheSize int) Watcher[*corev1.Event] {
	w := newWatcher[*corev1.Event](ctx, cacheSize)
	go func() {
		defer w.Close()

		for ev := range podEvents.Stream(w.ctx).Channel() {
			if ev.Error != nil {
				w.SendError(ev.Error)
			} else {
				w.SendValue(ev.Value)
				if ev.Value.Reason == "Scheduled" {
					return
				}
			}
		}
	}()
	return w
}

func WaitUntilContainerIsStarted(ctx context.Context, podEvents Watcher[*corev1.Event], containerName string) error {
	events := WatchContainerPreEvents(ctx, podEvents, containerName, 0, false)
	defer events.Close()

	for ev := range events.Stream(ctx).Channel() {
		if ev.Error != nil {
			return ev.Error
		} else if ev.Value.Reason == "Started" {
			return nil
		}
	}
	return ErrNoStartedEvent
}

func WatchContainerLogs(ctx context.Context, clientSet kubernetes.Interface, podEvents Watcher[*corev1.Event], namespace, podName, containerName string) Watcher[ContainerLog] {
	w := newWatcher[ContainerLog](ctx, 0)

	go func() {
		defer w.Close()

		// Wait until "Started" event, to avoid calling logs on the
		err := WaitUntilContainerIsStarted(ctx, podEvents, containerName)
		if err != nil {
			w.SendError(err)
			return
		}

		// Create logs stream request
		req := clientSet.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Follow:     true,
			Timestamps: true,
			Container:  containerName,
		})
		var stream io.ReadCloser
		for {
			stream, err = req.Stream(ctx)
			if err != nil {
				// The container is not necessarily already started when Started event is received
				if !strings.Contains(err.Error(), "is waiting to start") {
					w.SendError(err)
					return
				}
				continue
			}
			break
		}

		go func() {
			<-w.Done()
			_ = stream.Close()
		}()

		// Parse and return the logs
		reader := bufio.NewReader(stream)
		var tsPrefix, tmpTsPrefix []byte
		isNewLine := false
		isStarted := false
		var ts, tmpTs time.Time
		for {
			var prepend []byte

			// Read next timestamp
			tmpTs, tmpTsPrefix, err = ReadTimestamp(reader)
			if err == nil {
				ts = tmpTs
				tsPrefix = tmpTsPrefix
			} else if err == io.EOF {
				return
			} else {
				// Edge case: Kubernetes may send critical errors without timestamp (like ionotify)
				if len(tmpTsPrefix) > 0 {
					prepend = tmpTsPrefix
				}
				w.SendError(err)
			}

			// Check for the next part
			line, err := utils.ReadLongLine(reader)
			if len(prepend) > 0 {
				line = append(prepend, line...)
			}
			commentRe := regexp.MustCompile(fmt.Sprintf(`^%s(%s)?([^%s]+)%s([a-zA-Z0-9-_.]+)(?:%s([^\n]+))?%s$`,
				data.InstructionPrefix, data.HintPrefix, data.InstructionSeparator, data.InstructionSeparator, data.InstructionValueSeparator, data.InstructionSeparator))

			// Process the received line
			if len(line) > 0 {
				hadComment := false
				// Fast check to avoid regexes
				if len(line) >= 4 && string(line[:len(data.InstructionPrefix)]) == data.InstructionPrefix {
					v := commentRe.FindSubmatch(line)
					if v != nil {
						isHint := string(v[1]) == data.HintPrefix
						ref := string(v[2])
						name := string(v[3])
						result := Instruction{Ref: ref, Name: name}
						log := ContainerLog{Time: ts}
						if isHint {
							log.Hint = &result
						} else {
							log.Output = &result
						}
						if len(v) > 4 && v[4] != nil {
							err := json.Unmarshal(v[4], &result.Value)
							if err == nil {
								isNewLine = false
								hadComment = true
								w.SendValue(log)
							}
						} else {
							isNewLine = false
							hadComment = true
							w.SendValue(log)
						}
					}
				}

				// Append as regular log if expected
				if !hadComment {
					if isNewLine {
						line = append(append([]byte("\n"), tsPrefix...), line...)
					} else if !isStarted {
						line = append(tsPrefix, line...)
						isStarted = true
					}
					w.SendValue(ContainerLog{Time: ts, Log: line})
					isNewLine = true
				}
			} else if isStarted {
				w.SendValue(ContainerLog{Time: ts, Log: append([]byte("\n"), tsPrefix...)})
			}

			// Handle the error
			if err != nil {
				if err != io.EOF {
					w.SendError(err)
				}
				return
			}
		}
	}()

	return w
}

func ReadTimestamp(reader *bufio.Reader) (time.Time, []byte, error) {
	tsPrefix := make([]byte, 31) // 30 bytes for timestamp + 1 byte for space
	count, err := io.ReadFull(reader, tsPrefix)
	if err != nil {
		return time.Time{}, nil, err
	}
	if count < 31 {
		return time.Time{}, nil, io.EOF
	}
	ts, err := time.Parse(KubernetesLogTimeFormat, string(tsPrefix[0:30]))
	if err != nil {
		return time.Time{}, tsPrefix, errors2.Wrap(err, "parsing timestamp")
	}
	return ts, tsPrefix, nil
}
