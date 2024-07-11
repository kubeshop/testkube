package testworkflowcontroller

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/pkg/errors"
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
	FlushBufferSize = 65_536
	FlushLogTime    = 100 * time.Millisecond
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
func getContainerLogsStream(ctx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, follow bool, pod Channel[*corev1.Pod], since *time.Time) (io.Reader, error) {
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
		Follow:     follow,
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

func WatchContainerLogs(parentCtx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, follow bool, bufferSize int, pod Channel[*corev1.Pod]) Channel[ContainerLog] {
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
		stream, err := getContainerLogsStream(ctx, clientSet, namespace, podName, containerName, follow, pod, since)
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
		appendLog := func(ts time.Time, log ...[]byte) {
			logBufferMu.Lock()
			defer logBufferMu.Unlock()

			initialLogLen := logBufferLog.Len()
			if initialLogLen == 0 {
				logBufferTs = ts
			}
			for i := range log {
				logBufferLog.Write(log[i])
			}

			finalLogLen := logBufferLog.Len()
			flushable := finalLogLen > FlushLogMaxSize
			if flushable {
				unsafeFlushLogBuffer()
			}

			// Inform the flushing worker about a new log to flush.
			// Do it only when it's not scheduled
			if initialLogLen == 0 || flushable {
				select {
				case logBufferCh <- struct{}{}:
				default:
				}
			}
		}

		// Flush the log automatically after 100ms
		bufferCtx, bufferCtxCancel := context.WithCancel(ctx)
		defer bufferCtxCancel()
		go func() {
			t := time.NewTimer(FlushLogTime)
			for {
				t.Stop()

				if bufferCtx.Err() != nil {
					return
				}

				logLen := logBufferLog.Len()
				if logLen == 0 {
					select {
					case <-bufferCtx.Done():
						return
					case <-logBufferCh:
						continue
					}
				}

				t.Reset(FlushLogTime)
				select {
				case <-bufferCtx.Done():
					if !t.Stop() {
						<-t.C
					}
					return
				case <-t.C:
					flushLogBuffer()
				case <-logBufferCh:
					continue
				}
			}
		}()

		// Flush the rest of logs if it is closed
		defer flushLogBuffer()

		// Parse and return the logs
		reader := bufio.NewReaderSize(stream, FlushBufferSize)
		tsReader := newTimestampReader()
		isNewLine := false
		isStarted := false
		for {
			var prepend []byte

			// Read next timestamp
			err = tsReader.Read(reader)
			if err == nil {
				// Strip older logs - SinceTime in Kubernetes logs is ignoring milliseconds precision
				if since != nil && since.After(tsReader.ts) {
					_, _ = utils.ReadLongLine(reader)
					continue
				}
				hadAnyContent = true
			} else if err == io.EOF {
				if !hadAnyContent {
					return
				}
				// Reinitialize logs stream
				since = common.Ptr(tsReader.ts.Add(1))
				stream, err = getContainerLogsStream(ctx, clientSet, namespace, podName, containerName, follow, pod, since)
				if err != nil {
					return
				}
				reader.Reset(stream)
				hadAnyContent = false
				continue
			} else {
				// Edge case: Kubernetes may send critical errors without timestamp (like ionotify)
				if len(tsReader.Prefix()) > 0 {
					prepend = bytes.Clone(tsReader.Prefix())
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
					log := ContainerLog{Time: tsReader.ts}
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
						appendLog(tsReader.ts, tsReader.Prefix(), line)
						isStarted = true
					} else if isNewLine {
						appendLog(tsReader.ts, []byte("\n"), tsReader.Prefix(), line)
					}
					isNewLine = true
				}
			} else if isStarted {
				appendLog(tsReader.ts, []byte("\n"), tsReader.Prefix())
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

var (
	ErrInvalidTimestamp = errors.New("invalid timestamp")
)

type timestampReader struct {
	buffer []byte
	bytes  int
	ts     time.Time
	utc    *bool
}

func newTimestampReader() *timestampReader {
	return &timestampReader{
		buffer: make([]byte, 31, 36), // 30 bytes for timestamp + 1 byte for space + 5 additional bytes for non-UTC timezone
	}
}

func (t *timestampReader) Prefix() []byte {
	return t.buffer[:t.bytes]
}

// read is initial operation for reading the timestamp,
// that is the slowest one, but also detects the timestamp format.
// It's meant to be executed just once, for performance reasons.
func (t *timestampReader) read(reader *bufio.Reader) error {
	// Read the possible timestamp slice
	read, err := io.ReadFull(reader, t.buffer[:31])
	t.bytes = read
	if err != nil {
		return err
	}

	// Detect the timezone format and adjust the reader if needed
	utc := t.buffer[29] == 'Z'
	t.utc = &utc
	if !utc && len(t.buffer) < 35 {
		// Increase capacity to store the +00:00 time
		t.buffer = append(t.buffer, make([]byte, 5)...)

		// Read the missing part
		read, err = io.ReadFull(reader, t.buffer[31:])
		t.bytes += read
		if err != nil {
			return err
		}
	}

	// Compute the timestamp
	if utc {
		ts, err := time.Parse(time.RFC3339Nano, unsafe.String(&t.buffer[0], 30))
		if err != nil {
			return ErrInvalidTimestamp
		}
		t.ts = ts
	} else {
		ts, err := time.Parse(time.RFC3339Nano, unsafe.String(&t.buffer[0], 35))
		if err != nil {
			return ErrInvalidTimestamp
		}
		t.ts = ts.UTC()
	}
	return nil
}

// readUTC is optimized operation for reading the UTC timestamp (Z).
func (t *timestampReader) readUTC(reader *bufio.Reader) error {
	// Read the possible timestamp slice
	read, err := io.ReadFull(reader, t.buffer)
	t.bytes = read
	if err != nil {
		return err
	}

	// Compute the timestamp
	ts, err := time.Parse(time.RFC3339Nano, unsafe.String(&t.buffer[0], 30))
	if err != nil {
		return ErrInvalidTimestamp
	}
	t.ts = ts
	return nil
}

// readNonUTC is optimized operation for reading the non-UTC timestamp (+00:00).
func (t *timestampReader) readNonUTC(reader *bufio.Reader) error {
	// Read the possible timestamp slice
	read, err := io.ReadFull(reader, t.buffer)
	t.bytes = read
	if err != nil {
		return err
	}

	// Compute the timestamp
	ts, err := time.Parse(time.RFC3339Nano, unsafe.String(&t.buffer[0], 35))
	if err != nil {
		return ErrInvalidTimestamp
	}
	t.ts = ts.UTC()
	return nil
}

func (t *timestampReader) Read(reader *bufio.Reader) error {
	if t.utc == nil {
		return t.read(reader)
	} else if *t.utc {
		return t.readUTC(reader)
	}
	return t.readNonUTC(reader)
}
