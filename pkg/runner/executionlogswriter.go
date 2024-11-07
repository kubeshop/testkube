package runner

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/pkg/errors"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/bufferedstream"
)

type LogPresigner interface {
	PresignSaveLog(ctx context.Context, id string, workflowName string) (string, error)
}

type ExecutionLogsWriter interface {
	io.Writer
	WriteStart(ref string) error
	Save(ctx context.Context) error
	Saved() bool
	Cleanup()
	Reset() error
}

type executionLogsWriter struct {
	presigner    LogPresigner
	id           string
	workflowName string
	skipVerify   bool

	writer *io.PipeWriter
	buffer bufferedstream.BufferedStream
	mu     sync.Mutex
}

func NewExecutionLogsWriter(presigner LogPresigner, id string, workflowName string, skipVerify bool) (ExecutionLogsWriter, error) {
	e := &executionLogsWriter{
		presigner:    presigner,
		id:           id,
		workflowName: workflowName,
		skipVerify:   skipVerify,
	}
	err := e.Reset()
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *executionLogsWriter) Write(p []byte) (n int, err error) {
	return e.writer.Write(p)
}

func (e *executionLogsWriter) WriteStart(ref string) error {
	_, err := e.Write([]byte(instructions.SprintHint(ref, initconstants.InstructionStart)))
	return err
}

func (e *executionLogsWriter) Save(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.buffer == nil {
		return errors.New("the writer is already cleaned up")
	}

	e.writer.Close()

	url, err := e.presigner.PresignSaveLog(ctx, e.id, e.workflowName)
	if err != nil {
		return err
	}

	content := e.buffer
	contentLen := content.Len()
	if contentLen == 0 {
		content = nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, content)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/octet-stream")
	req.ContentLength = int64(contentLen)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to save file in the object storage")
	}
	if res.StatusCode != http.StatusOK {
		return errors.Errorf("error saving file with presigned url: expected 200 OK response code, got %d", res.StatusCode)
	}
	e.cleanup()
	return nil
}

func (e *executionLogsWriter) cleanup() {
	if e.writer != nil {
		e.writer.Close()
	}
	if e.buffer != nil {
		e.buffer.Cleanup()
	}
}

func (e *executionLogsWriter) Cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cleanup()
}

func (e *executionLogsWriter) Saved() bool {
	if v := e.mu.TryLock(); v {
		defer e.mu.Unlock()
		return e.buffer == nil
	}
	return false
}

func (e *executionLogsWriter) Reset() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cleanup()

	r, writer := io.Pipe()
	reader := bufio.NewReader(r)
	// TODO: consider how to choose temp dir
	buffer, err := bufferedstream.NewBufferedStream("", "log", reader)
	if err != nil {
		return err
	}
	e.writer = writer
	e.buffer = buffer
	return nil
}
