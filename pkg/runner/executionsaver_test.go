package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
)

type failingExecutionLogsWriter struct {
	enabled    bool
	saveErr    error
	saved      bool
	saveCalled bool
}

func (w *failingExecutionLogsWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *failingExecutionLogsWriter) WriteStart(string) error {
	return nil
}

func (w *failingExecutionLogsWriter) Save(context.Context) error {
	w.saveCalled = true
	if w.saveErr != nil {
		return w.saveErr
	}
	w.saved = true
	return nil
}

func (w *failingExecutionLogsWriter) Saved() bool {
	return w.saved
}

func (w *failingExecutionLogsWriter) Enabled() bool {
	return w.enabled
}

func (w *failingExecutionLogsWriter) Cleanup() {}

func (w *failingExecutionLogsWriter) Reset() error {
	w.saved = false
	return nil
}

func TestExecutionSaverEnd_SkipsLogSaveWhenStorageDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := controlplaneclient.NewMockClient(ctrl)
	logs := &failingExecutionLogsWriter{}
	saver, err := NewExecutionSaver(context.Background(), client, "execution-id", "org-id", "env-id", "runner-id", logs)
	require.NoError(t, err)

	status := testkube.PASSED_TestWorkflowStatus
	result := testkube.TestWorkflowResult{Status: &status}

	client.EXPECT().
		FinishExecutionResult(gomock.Any(), "env-id", "execution-id", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, got *testkube.TestWorkflowResult) error {
			require.Same(t, &status, got.Status)
			return nil
		})

	err = saver.End(context.Background(), result)
	require.NoError(t, err)
	require.False(t, logs.saveCalled)
}

func TestExecutionSaverEnd_LogSaveFailureBlocksFinalizationWhenStorageEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := controlplaneclient.NewMockClient(ctrl)
	logs := &failingExecutionLogsWriter{enabled: true, saveErr: errors.New("object storage unavailable")}
	saver, err := NewExecutionSaver(context.Background(), client, "execution-id", "org-id", "env-id", "runner-id", logs)
	require.NoError(t, err)

	status := testkube.PASSED_TestWorkflowStatus
	result := testkube.TestWorkflowResult{Status: &status}

	err = saver.End(context.Background(), result)
	require.ErrorContains(t, err, "object storage unavailable")
	require.True(t, logs.saveCalled)
}

func TestExecutionSaverEnd_OutputSaveFailureBlocksFinalization(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := controlplaneclient.NewMockClient(ctrl)
	logs := &failingExecutionLogsWriter{enabled: true}
	saver, err := NewExecutionSaver(context.Background(), client, "execution-id", "org-id", "env-id", "runner-id", logs)
	require.NoError(t, err)

	saver.AppendOutput(testkube.TestWorkflowOutput{Name: "artifact"})
	status := testkube.PASSED_TestWorkflowStatus
	result := testkube.TestWorkflowResult{Status: &status}

	client.EXPECT().
		UpdateExecutionOutput(gomock.Any(), "env-id", "execution-id", []testkube.TestWorkflowOutput{{Name: "artifact"}}).
		Return(errors.New("database unavailable"))

	err = saver.End(context.Background(), result)
	require.ErrorContains(t, err, "database unavailable")
}
