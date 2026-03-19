package none

import (
	"bytes"
	"context"
	"io"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

var _ testworkflow.OutputRepository = (*NoneRepository)(nil)

// NoneRepository is a no-op implementation of OutputRepository that discards all log data.
// It is used when log persistence is disabled (logs.storage: "none").
type NoneRepository struct{}

func NewNoneOutputRepository() *NoneRepository {
	return &NoneRepository{}
}

func (n *NoneRepository) PresignSaveLog(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (n *NoneRepository) PresignReadLog(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (n *NoneRepository) SaveLog(_ context.Context, _, _ string, reader io.Reader) error {
	_, err := io.Copy(io.Discard, reader)
	return err
}

func (n *NoneRepository) ReadLog(_ context.Context, _, _ string) (io.ReadCloser, error) {
	return io.NopCloser(&bytes.Reader{}), nil
}

func (n *NoneRepository) HasLog(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (n *NoneRepository) DeleteOutputByTestWorkflow(_ context.Context, _ string) error {
	return nil
}

func (n *NoneRepository) DeleteOutputForTestWorkflows(_ context.Context, _ []string) error {
	return nil
}
