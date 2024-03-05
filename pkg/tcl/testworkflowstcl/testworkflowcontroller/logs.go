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
	"github.com/kubeshop/testkube/pkg/utils"
)

type Hint struct {
	Ref   string
	Name  string
	Value interface{}
}

type Comment struct {
	Time   time.Time
	Hint   *Hint
	Output *Hint
}

type ContainerLog struct {
	Time   time.Time
	Log    []byte
	Hint   *Hint
	Output *Hint
}

type ContainerResult struct {
	Status   string
	ExitCode int
	Took     time.Duration
}

var UnknownContainerResult = ContainerResult{
	Status:   "unknown",
	ExitCode: -1,
}

func GetContainerResult(ctx context.Context, pod Watcher[*corev1.Pod], containerName string) (ContainerResult, error) {
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
		re := regexp.MustCompile(`^([^,]*),(0|[1-9]\d*)$`)
		msg := c.Value.State.Terminated.Message
		match := re.FindStringSubmatch(msg)
		if match == nil {
			return UnknownContainerResult, fmt.Errorf("invalid termination message: %s", msg)
		}
		status := match[1]
		exitCode, _ := strconv.Atoi(match[2])
		if status == "" {
			status = "passed"
		}
		took := c.Value.State.Terminated.FinishedAt.Sub(c.Value.State.Terminated.StartedAt.Time)
		return ContainerResult{Status: status, ExitCode: exitCode, Took: took}, nil
	}
	return UnknownContainerResult, nil
}

var ErrNoStartedEvent = errors.New("started event not received")

func WatchContainerPreEvents(ctx context.Context, podEvents Watcher[*corev1.Event], containerName string, cacheSize int) Watcher[*corev1.Event] {
	w := newWatcher[*corev1.Event](ctx, cacheSize)
	go func() {
		events := WatchContainerEvents(ctx, podEvents, containerName, 0)
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

func WaitUntilContainerIsStarted(ctx context.Context, podEvents Watcher[*corev1.Event], containerName string) error {
	events := WatchContainerPreEvents(ctx, podEvents, containerName, 0)
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
		var tsPrefix []byte
		isNewLine := false
		isStarted := false
		var ts time.Time
		for {
			// Read next timestamp
			ts, tsPrefix, err = ReadTimestamp(reader)
			if err != nil {
				if err != io.EOF {
					w.SendError(err)
				}
				return
			}

			// Check for the next part
			line, err := utils.ReadLongLine(reader)
			commentRe := regexp.MustCompile(fmt.Sprintf(`^%s(%s)?([^;]+);?([a-zA-Z0-9_]+)(?::(.+))?;$`, data.InstructionPrefix, data.HintPrefix))

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
						result := Hint{Ref: ref, Name: name}
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
	ts, err := time.Parse(time.RFC3339Nano, string(tsPrefix[0:30]))
	if err != nil {
		return time.Time{}, nil, errors2.Wrap(err, "parsing timestamp")
	}
	return ts, tsPrefix, nil
}

func ReadHintOrOutput(reader *bufio.Reader) *Comment {
	// TODO: Read timestamp too
	next, err := reader.Peek(1)
	if err != nil || string(next) != ";" {
		return nil
	}
	next, err = reader.Peek(2)
	if err != nil || string(next) != ";;" {
		return nil
	}
	next, err = reader.Peek(3)
	if err != nil {
		return nil
	}

	if string(next) == ";;;" {
		// Could be hint
	}

	return nil
}
