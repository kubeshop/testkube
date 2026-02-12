package controller

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	FlushLogMaxSize = 100_000
	FlushBufferSize = 65_536
	FlushLogTime    = 100 * time.Millisecond

	LogRetryOnConnectionLostDelay  = 300 * time.Millisecond
	LogRetryOnWaitingForStartDelay = 100 * time.Millisecond
	LogStreamIdleTimeout           = 30 * time.Second

	LogRetryMaxAttempts            = 10
	LogProxyErrorRetryInitialDelay = 500 * time.Millisecond
	LogProxyErrorRetryMaxDelay     = 5 * time.Second
)

type Comment struct {
	Time   time.Time
	Hint   *instructions.Instruction
	Output *instructions.Instruction
}

type ContainerLog struct {
	Time   time.Time
	Log    []byte
	Hint   *instructions.Instruction
	Output *instructions.Instruction
}

type ContainerLogType string

const (
	ContainerLogTypeHint   ContainerLogType = "hint"
	ContainerLogTypeOutput ContainerLogType = "output"
	ContainerLogTypeLog    ContainerLogType = ""
)

func (c *ContainerLog) Type() ContainerLogType {
	if c.Hint != nil {
		return ContainerLogTypeHint
	} else if c.Output != nil {
		return ContainerLogTypeOutput
	}
	return ContainerLogTypeLog
}

// getContainerLogsStream is getting logs stream, and tries to reinitialize the stream on EOF.
// EOF may happen not only on the actual container end, but also in case of the log rotation.
// @see {@link https://stackoverflow.com/a/68673451}
func getContainerLogsStream(ctx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, isDone func() bool, since *time.Time) (io.Reader, error) {
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
		Follow:     !isDone(),
		Timestamps: true,
		SinceTime:  sinceTime,
	})
	var err error
	var stream io.ReadCloser
	retries := 0
	for {
		stream, err = req.Stream(ctx)
		if err == nil {
			return stream, nil
		}

		errMsg := err.Error()
		var delay time.Duration
		switch {
		case strings.Contains(errMsg, "connection lost"):
			retries++
			if retries > LogRetryMaxAttempts {
				return nil, err
			}
			delay = LogRetryOnConnectionLostDelay
			log.DefaultLogger.Warnw("connection lost while loading container logs, retrying", "pod", podName, "attempt", retries, "error", err)
		case strings.Contains(errMsg, "tls: internal error"):
			retries++
			if retries > LogRetryMaxAttempts {
				return nil, err
			}
			delay = LogRetryOnConnectionLostDelay
			log.DefaultLogger.Errorw("cluster's TLS error (likely CSR signing delay) while loading container logs, retrying", "pod", podName, "attempt", retries, "error", err)
		case strings.Contains(errMsg, "proxy error"):
			retries++
			if retries > LogRetryMaxAttempts {
				return nil, err
			}
			delay = LogProxyErrorRetryInitialDelay << (retries - 1)
			if delay > LogProxyErrorRetryMaxDelay {
				delay = LogProxyErrorRetryMaxDelay
			}
			log.DefaultLogger.Warnw("proxy error while loading container logs, retrying", "pod", podName, "attempt", retries, "delay", delay, "error", err)
		case strings.Contains(errMsg, "is waiting to start"):
			if isDone() {
				return bytes.NewReader(nil), io.EOF
			}
			delay = LogRetryOnWaitingForStartDelay
		default:
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
}

type logStreamOpener func(ctx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, isDone func() bool, since *time.Time) (io.Reader, error)

func WatchContainerLogs(parentCtx context.Context, clientSet kubernetes.Interface, namespace, podName, containerName string, bufferSize int, isDone func() bool, isLastHint func(*instructions.Instruction) bool) <-chan ChannelMessage[ContainerLog] {
	return watchContainerLogsWithStream(parentCtx, getContainerLogsStream, clientSet, namespace, podName, containerName, bufferSize, isDone, isLastHint, LogStreamIdleTimeout)
}

func watchContainerLogsWithStream(parentCtx context.Context, opener logStreamOpener, clientSet kubernetes.Interface, namespace, podName, containerName string, bufferSize int, isDone func() bool, isLastHint func(*instructions.Instruction) bool, idleTimeout time.Duration) <-chan ChannelMessage[ContainerLog] {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	ch := make(chan ChannelMessage[ContainerLog], bufferSize)
	var mu sync.Mutex
	var lastActivity int64
	atomic.StoreInt64(&lastActivity, time.Now().UnixNano())
	touch := func() {
		atomic.StoreInt64(&lastActivity, time.Now().UnixNano())
	}

	sendError := func(err error) {
		defer func() {
			recover() // ignore already closed
			mu.Unlock()
		}()

		mu.Lock()
		ch <- ChannelMessage[ContainerLog]{Error: err}
	}

	sendLog := func(log ContainerLog) {
		defer func() {
			recover() // ignore already closed
			mu.Unlock()
		}()

		mu.Lock()
		ch <- ChannelMessage[ContainerLog]{Value: log}
	}

	go func() {
		<-ctx.Done()
		close(ch)
	}()

	if idleTimeout > 0 {
		go func() {
			ticker := time.NewTicker(idleTimeout)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if !isDone() {
						continue
					}
					last := time.Unix(0, atomic.LoadInt64(&lastActivity))
					if time.Since(last) >= idleTimeout {
						sendError(fmt.Errorf("log stream idle timeout after %s", idleTimeout))
						ctxCancel()
						return
					}
				}
			}
		}()
	}

	go func() {
		defer ctxCancel()
		var err error

		var since *time.Time

		// Create logs stream request
		stream, err := opener(ctx, clientSet, namespace, podName, containerName, isDone, since)
		if err == io.EOF {
			return
		} else if err != nil {
			if !errors.Is(err, context.Canceled) {
				sendError(err)
			}
			return
		}

		// Build a buffer for logs to avoid scheduling Log notification for each write
		var logBufferLog bytes.Buffer
		var logBufferTs time.Time
		var logBufferMu sync.Mutex
		var logBufferCh = make(chan struct{}, 1)
		unsafeFlushLogBuffer := func() {
			if logBufferLog.Len() == 0 || ctx.Err() != nil {
				return
			}
			message := make([]byte, logBufferLog.Len())
			_, err := logBufferLog.Read(message)
			if err != nil {
				log.DefaultLogger.Errorf("failed to read log buffer: %s/%s", podName, containerName)
				return
			}
			sendLog(ContainerLog{Time: logBufferTs, Log: message})
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
		defer func() {
			bufferCtxCancel()
		}()
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
					t.Stop()

					return
				case <-t.C:
					flushLogBuffer()
				case <-logBufferCh:
					continue
				}
			}
		}()

		// Flush the rest of logs if it is closed
		defer func() {
			flushLogBuffer()
		}()

		// Parse and return the logs
		reader := bufio.NewReaderSize(stream, FlushBufferSize)
		readerAnyContent := false
		tsReader := newTimestampReader()
		lastTs := time.Now()
		completed := false

		hasNewLine := false

		for {
			// --- Step 1: READING TIMESTAMP

			// Read next timestamp
			err = tsReader.Read(reader)
			if err == nil || errors.Is(err, ErrInvalidTimestamp) {
				touch()
			}

			// Handle context canceled
			if errors.Is(err, context.Canceled) {
				return
			}

			// Ignore too old logs. SinceTime in Kubernetes is precise only to seconds
			if err == nil && !readerAnyContent {
				if since != nil && since.After(tsReader.ts) {
					isPrefix := true
					for isPrefix && err == nil {
						_, isPrefix, err = reader.ReadLine()
					}
					continue
				}
				readerAnyContent = true
			}

			// Save information about the last timestamp
			if err == nil {
				lastTs = tsReader.ts
			}

			// If the stream is finished,
			// either the logfile has been rotated, or the container actually finished.
			// Consider the container is done only when either:
			// - there was EOF without any logs since, or
			// - the last expected instruction was already delivered
			if err == io.EOF && (!readerAnyContent || completed) {
				return
			}

			// If there was EOF, and we are not sure if container is done,
			// reinitialize the stream from the time we have finished.
			// Similarly for GOAWAY, that may be caused by too long connection.
			if err == io.EOF || (err != nil && strings.Contains(err.Error(), "GOAWAY")) {
				since = common.Ptr(lastTs.Add(1))
				stream, err = opener(ctx, clientSet, namespace, podName, containerName, isDone, since)
				if err != nil {
					return
				}
				reader.Reset(stream)
				readerAnyContent = false
				continue
			}

			// Edge case: Kubernetes may send critical errors without timestamp (like ionotify)
			if errors.Is(err, ErrInvalidTimestamp) && len(tsReader.Prefix()) > 0 {
				appendLog(lastTs, []byte(tsReader.Format(lastTs)), []byte(" "), tsReader.Prefix())
				rest, _ := utils.ReadLongLine(reader)
				appendLog(lastTs, rest, []byte("\n"))
				hasNewLine = false
				continue
			}

			// Push information about any other error
			if err != nil {
				sendError(err)
				continue
			}

			// --- Step 2: READING THE BEGINNING OF THE LINE

			line, isPrefix, err := reader.ReadLine()

			// Between instructions there may be empty line that should be just ignored
			if !isPrefix && len(line) == 0 {
				if hasNewLine {
					appendLog(lastTs, []byte("\n"))
				}
				continue
			}

			// Fast-track: we know this line won't be an instruction
			if !instructions.MayBeInstruction(line) {
				if hasNewLine {
					appendLog(lastTs, []byte("\n"))
				}
				appendLog(lastTs, tsReader.Prefix(), line)
				for isPrefix && err == nil {
					line, isPrefix, err = reader.ReadLine()
					appendLog(lastTs, line)
				}
				hasNewLine = true
				continue
			}

			// --- Step 3: FINISH READING THE LINE AND EXPORT DATA

			// Ensure we read the whole line to buffer to validate if it is instruction
			for isPrefix && err == nil {
				var currentLine []byte
				currentLine, isPrefix, err = reader.ReadLine()
				line = append(line, currentLine...)
			}

			// Detect instruction
			instruction, isHint, err := instructions.DetectInstruction(line)
			if err == nil && instruction != nil {
				item := ContainerLog{Time: lastTs}
				if isHint {
					item.Hint = instruction
					if !completed && isLastHint(instruction) {
						completed = true
					}
				} else {
					item.Output = instruction
				}
				flushLogBuffer()
				sendLog(item)
				hasNewLine = false
				continue
			}

			// Print line if it's not an instruction
			if hasNewLine {
				appendLog(lastTs, []byte("\n"))
			}
			appendLog(lastTs, tsReader.Prefix(), line)
			hasNewLine = true
		}
	}()

	return ch
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

func (t *timestampReader) Format(ts time.Time) string {
	if t.utc == nil || *t.utc {
		return ts.Format(constants.PreciseTimeFormat)
	}
	return ts.Format(KubernetesTimezoneLogTimeFormat)
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
