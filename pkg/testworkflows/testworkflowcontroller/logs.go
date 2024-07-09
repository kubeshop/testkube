package testworkflowcontroller

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	FlushLogMaxSize = 65_536
	FlushLogTime    = 50 * time.Millisecond
	FlushLogMaxTime = 100 * time.Millisecond
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

func WatchContainerLogs(ctx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, bufferSize int, follow bool, pod Channel[*corev1.Pod]) Channel[ContainerLog] {
	w := newChannel[ContainerLog](ctx, bufferSize)

	go func() {
		defer w.Close()
		var err error

		// Create logs stream request
		req := clientSet.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Follow:     follow,
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
				p := <-pod.Peek(ctx)
				if p != nil && IsPodDone(p) {
					w.Error(errors.New("pod is finished and there are no logs for this container"))
				}
				continue
			}
			break
		}

		go func() {
			<-w.Done()
			_ = stream.Close()
		}()

		// Build a buffer for logs to avoid scheduling Log notification for each write
		var logBufferLog bytes.Buffer
		var logBufferTs time.Time
		var logBufferMu sync.Mutex
		var logBufferCh = make(chan struct{}, 1)
		defer close(logBufferCh)
		unsafeFlushLogBuffer := func() {
			if logBufferLog.Len() == 0 {
				return
			}
			message := make([]byte, logBufferLog.Len())
			_, err := logBufferLog.Read(message)
			if err != nil {
				log.DefaultLogger.Errorf("failed to read log buffer: %s/%s", podName, containerName)
				return
			}
			w.Send(ContainerLog{Time: logBufferTs, Log: message})

		}
		flushLogBuffer := func() {
			logBufferMu.Lock()
			defer logBufferMu.Unlock()
			unsafeFlushLogBuffer()
		}
		appendLog := func(ts time.Time, log []byte) {
			if len(log) == 0 {
				return
			}
			logBufferMu.Lock()
			defer logBufferMu.Unlock()
			if logBufferLog.Len() == 0 {
				logBufferTs = ts
			}
			logBufferLog.Write(log)

			// Inform the flushing worker about a new log to flush
			select {
			case logBufferCh <- struct{}{}:
			default:
			}
		}
		defer flushLogBuffer()

		// Flush the log automatically when expected
		bufferCtx, bufferCtxCancel := context.WithCancel(ctx)
		defer bufferCtxCancel()
		go func() {
			flushLogTimer := time.NewTimer(FlushLogMaxTime)
			flushLogTimerEnabled := false

			for {
				if bufferCtx.Err() != nil {
					return
				}

				logLen := logBufferLog.Len()

				if logLen > FlushLogMaxSize {
					flushLogBuffer()
					continue
				}

				if logLen == 0 {
					flushLogTimerEnabled = false
					select {
					case <-bufferCtx.Done():
						return
					case <-logBufferCh:
						continue
					}
				}

				if !flushLogTimerEnabled {
					flushLogTimerEnabled = true
					flushLogTimer.Reset(FlushLogMaxTime)
				}

				select {
				case <-bufferCtx.Done():
					return
				case <-flushLogTimer.C:
					flushLogBuffer()
				case <-time.After(FlushLogTime):
					flushLogBuffer()
				case <-logBufferCh:
					continue
				}
			}
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
				flushLogBuffer()
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
					flushLogBuffer()
					w.Send(log)
				}

				// Append as regular log if expected
				if !hadComment {
					if !isStarted {
						appendLog(ts, tsPrefix)
						appendLog(ts, line)
						isStarted = true
					} else if isNewLine {
						appendLog(ts, []byte("\n"))
						appendLog(ts, tsPrefix)
						appendLog(ts, line)
					}
					isNewLine = true
				}
			} else if isStarted {
				appendLog(ts, []byte("\n"))
				appendLog(ts, tsPrefix)
			}

			// Handle the error
			if err != nil {
				if err != io.EOF {
					flushLogBuffer()
					w.Error(err)
				}
				return
			}
		}
	}()

	return w
}

func ReadTimestamp(reader *bufio.Reader) (time.Time, []byte, error) {
	tsPrefix := make([]byte, 31, 35) // 30 bytes for timestamp + 1 byte for space + 4 additional bytes for non-UTC timezone
	count, err := io.ReadFull(reader, tsPrefix)
	if err != nil {
		return time.Time{}, nil, err
	}
	if count < 31 {
		return time.Time{}, nil, io.EOF
	}
	var ts time.Time
	// Handle non-UTC timezones
	if tsPrefix[29] == '+' {
		tsSuffix := make([]byte, 5)
		count, err = io.ReadFull(reader, tsSuffix)
		if err != nil {
			return time.Time{}, nil, err
		}
		if count < 5 {
			return time.Time{}, nil, io.EOF
		}
		tsPrefix = append(tsPrefix, tsSuffix...)
	}
	ts, err = time.Parse(KubernetesTimezoneLogTimeFormat, string(tsPrefix[0:len(tsPrefix)-1]))
	if err != nil {
		return time.Time{}, tsPrefix, errors2.Wrap(err, "parsing timestamp")
	}
	return ts.UTC(), tsPrefix, nil
}
