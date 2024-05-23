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
	"io"
	"strings"
	"time"

	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/utils"
)

type Comment struct {
	Time   time.Time
	Hint   *data.Instruction
	Output *data.Instruction
}

type ContainerLog struct {
	Time   time.Time
	Log    []byte
	Hint   *data.Instruction
	Output *data.Instruction
}

func WatchContainerLogs(ctx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, bufferSize int) Channel[ContainerLog] {
	w := newChannel[ContainerLog](ctx, bufferSize)

	go func() {
		defer w.Close()
		var err error

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
					w.Error(err)
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
				w.Error(err)
			}

			// Check for the next part
			line, err := utils.ReadLongLine(reader)
			if len(prepend) > 0 {
				line = append(prepend, line...)
			}

			// Process the received line
			if !isNewLine && len(line) == 0 {
				isNewLine = true
			} else if len(line) > 0 {
				hadComment := false
				instruction, isHint, err := data.DetectInstruction(line)
				if err == nil && instruction != nil {
					isNewLine = false
					hadComment = true
					log := ContainerLog{Time: ts}
					if isHint {
						log.Hint = instruction
					} else {
						log.Output = instruction
					}
					w.Send(log)
				}

				// Append as regular log if expected
				if !hadComment {
					if !isStarted {
						line = append(tsPrefix, line...)
						isStarted = true
					} else if isNewLine {
						line = append(append([]byte("\n"), tsPrefix...), line...)
					}
					w.Send(ContainerLog{Time: ts, Log: line})
					isNewLine = true
				}
			} else if isStarted {
				w.Send(ContainerLog{Time: ts, Log: append([]byte("\n"), tsPrefix...)})
			}

			// Handle the error
			if err != nil {
				if err != io.EOF {
					w.Error(err)
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
