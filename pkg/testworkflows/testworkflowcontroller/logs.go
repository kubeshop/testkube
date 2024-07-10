package testworkflowcontroller

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"time"

	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	FlushLogMaxSize = 100_000
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

// getContainerLogsStream is getting logs stream, and tries to reinitialize the stream on EOF.
// EOF may happen not only on the actual container end, but also in case of the log rotation.
// @see {@link https://stackoverflow.com/a/68673451}
func getContainerLogsStream(ctx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, pod Channel[*corev1.Pod], since *time.Time) (io.Reader, error) {
	// Fail immediately if the context is finished
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Build Kubernetes structure for time
	var sinceTime *metav1.Time
	if since != nil {
		sinceTime = &metav1.Time{Time: *since}
	}

	// Create logs stream request
	req := clientSet.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		Timestamps: true,
		SinceTime:  sinceTime,
	})
	var err error
	var stream io.ReadCloser
	for {
		stream, err = req.Stream(ctx)
		if err != nil {
			// The container is not necessarily already started when Started event is received
			if !strings.Contains(err.Error(), "is waiting to start") {
				return nil, err
			}
			p := <-pod.Peek(ctx)
			if p == nil {
				return bytes.NewReader(nil), io.EOF
			}
			containerDone := IsPodDone(p)
			for i := range p.Status.InitContainerStatuses {
				if p.Status.InitContainerStatuses[i].Name == containerName {
					if p.Status.InitContainerStatuses[i].State.Terminated != nil {
						containerDone = true
						break
					}
				}
			}
			for i := range p.Status.ContainerStatuses {
				if p.Status.ContainerStatuses[i].Name == containerName {
					if p.Status.ContainerStatuses[i].State.Terminated != nil {
						containerDone = true
						break
					}
				}
			}

			if containerDone {
				return bytes.NewReader(nil), io.EOF
			}
			continue
		}
		break
	}
	return stream, nil
}

func WatchContainerLogs(parentCtx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, bufferSize int, pod Channel[*corev1.Pod]) Channel[ContainerLog] {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	w := newChannel[ContainerLog](ctx, bufferSize)

	go func() {
		<-w.Done()
		ctxCancel()
	}()

	go func() {
		defer ctxCancel()
		var err error

		var since *time.Time

		// Create logs stream request
		stream, err := getContainerLogsStream(ctx, clientSet, namespace, podName, containerName, pod, since)
		hadAnyContent := false
		if err == io.EOF {
			return
		} else if err != nil {
			w.Error(err)
			return
		}

		// Build a buffer for logs to avoid scheduling Log notification for each write
		var logBufferLog bytes.Buffer
		var logBufferTs time.Time
		var logBufferMu sync.Mutex
		var logBufferCh = make(chan struct{}, 1)
		unsafeFlushLogBuffer := func() {
			if logBufferLog.Len() == 0 || w.CtxErr() != nil {
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

		// Flush the rest of logs if it is closed
		defer flushLogBuffer()

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
				// Strip older logs - SinceTime in Kubernetes logs is ignoring milliseconds precision
				if since != nil && since.After(tmpTs) {
					_, _ = utils.ReadLongLine(reader)
					continue
				}

				ts = tmpTs
				tsPrefix = tmpTsPrefix
				hadAnyContent = true
			} else if err == io.EOF {
				if !hadAnyContent {
					return
				}
				// Reinitialize logs stream
				since = common.Ptr(ts.Add(1))
				stream, err = getContainerLogsStream(ctx, clientSet, namespace, podName, containerName, pod, since)
				if err != nil {
					return
				}
				reader = bufio.NewReader(stream)
				hadAnyContent = false
				continue
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
