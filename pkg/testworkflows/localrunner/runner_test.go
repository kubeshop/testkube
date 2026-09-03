package localrunner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

func TestNewRunIDIsKubernetesSafeAndUnique(t *testing.T) {
	first := newRunID("Demo workflow / unsafe")
	second := newRunID("Demo workflow / unsafe")
	assert.NotEqual(t, first, second)
	assert.LessOrEqual(t, len(first), 63)
	assert.True(t, strings.HasPrefix(first, "local-demo-workflow-unsafe-"))
	_, err := Labels(first, "workflow")
	require.NoError(t, err)
}

func TestLocalWorkerConfigUsesSafeIdentityForLongPlaywrightFailureRun(t *testing.T) {
	// This has the shape and length of the real failure fixture's generated
	// local run ID. Adding the historical worker "local-" prefix made the
	// runner label exceed the Kubernetes 63-byte label-value limit.
	const runID = "local-local-runner-playwright-artifact-failure-e3c8d96f3008"

	config, err := localWorkerConfig(DefaultNamespace, runID)
	require.NoError(t, err)
	repeated, err := localWorkerConfig(DefaultNamespace, runID)
	require.NoError(t, err)

	assert.Equal(t, config.Cluster.Id, config.RunnerId)
	assert.Equal(t, config.RunnerId, repeated.RunnerId)
	assert.LessOrEqual(t, len(config.RunnerId), validation.LabelValueMaxLength)
	assert.Empty(t, validation.IsValidLabelValue(config.RunnerId))
	assert.NotEqual(t, "local-"+runID, config.RunnerId)
}

func TestLocalLogAndResultHelpers(t *testing.T) {
	assert.Equal(t, "hello", trimLocalTimestamp("2026-09-03T10:11:12Z hello"))
	assert.Equal(t, "plain", trimLocalTimestamp("plain"))
	passed := testkube.PASSED_TestWorkflowStatus
	result := &testkube.TestWorkflowResult{Status: &passed}
	assert.Equal(t, "passed", resultStatus(result))
	assert.Equal(t, "unknown", resultStatus(nil))
	assert.True(t, isProtocolNotification("ready"))
	assert.False(t, isProtocolNotification("log"))
}

func TestRenderLogRedactsLocalSourceURLAndToken(t *testing.T) {
	source := &SourcePlan{URL: "http://local-source:8080/0123456789abcdef.tar.gz"}
	var output bytes.Buffer
	renderLogRedacted(&output, "Downloading http://local-source:8080/0123456789abcdef.tar.gz and token 0123456789abcdef\n", localLogRedactions(source))
	assert.Equal(t, "Downloading <local-source> and token <local-source>\n", output.String())
	assert.NotContains(t, output.String(), "0123456789abcdef")
}

func TestRenderLogRedactsRuntimeSourceURLWithoutPreparedSource(t *testing.T) {
	var output bytes.Buffer
	renderLogRedacted(&output, "Downloading http://local-source-local-run-012345:8080/abcdef0123456789.tar.gz\n", nil)
	assert.Equal(t, "Downloading <local-source>\n", output.String())
}

func TestRenderLogRedactsLocalArtifactEndpointAndToken(t *testing.T) {
	artifactRelay := &ArtifactRelay{
		URL:   "http://local-artifacts-local-run-012345:8080/upload",
		token: "artifact-relay-token",
	}
	var output bytes.Buffer
	renderLogRedacted(&output, "Uploading to http://local-artifacts-local-run-012345:8080/upload with artifact-relay-token\n", localLogRedactions(nil, artifactRelay))
	assert.Equal(t, "Uploading to <local-artifact>/upload with <local-source>\n", output.String())
	assert.NotContains(t, output.String(), "artifact-relay-token")
}

func TestRedactLocalSourceErrorPreservesCauseAndHidesRelayCredentials(t *testing.T) {
	const (
		token = "abcdef0123456789"
		url   = "http://local-source-local-run-012345:8080/abcdef0123456789.tar.gz"
	)
	cause := errors.New("remote command failed for /srv/" + token + ".tar.gz from " + url)
	err := redactLocalSourceError(cause, url, token)
	assert.ErrorIs(t, err, cause)
	assert.NotContains(t, err.Error(), token)
	assert.NotContains(t, err.Error(), url)
	assert.Contains(t, err.Error(), "<local-source>")
}

func TestRedactLocalArtifactErrorPreservesCauseAndHidesRelayCredentials(t *testing.T) {
	const (
		token = "abcdef0123456789"
		url   = "http://local-artifacts-local-run-012345:8080"
	)
	cause := errors.New("artifact relay request failed for " + url + " with token " + token)
	err := redactLocalArtifactError(cause, url, token)
	assert.ErrorIs(t, err, cause)
	assert.NotContains(t, err.Error(), token)
	assert.NotContains(t, err.Error(), url)
	assert.Contains(t, err.Error(), "<local-artifact>")
}

func TestLocalJobOutcomeUsesKubernetesTerminalStatus(t *testing.T) {
	client := fake.NewClientset(
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "passed", Namespace: DefaultNamespace}, Status: batchv1.JobStatus{Succeeded: 1, Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}}},
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "failed", Namespace: DefaultNamespace}, Status: batchv1.JobStatus{Failed: 1, Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}}}},
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "retrying", Namespace: DefaultNamespace}, Status: batchv1.JobStatus{Failed: 1}},
	)
	finished, passed, status, err := localJobOutcome(context.Background(), client, DefaultNamespace, "passed")
	require.NoError(t, err)
	assert.True(t, finished)
	assert.True(t, passed)
	assert.Equal(t, "passed", status)
	finished, passed, status, err = localJobOutcome(context.Background(), client, DefaultNamespace, "failed")
	require.NoError(t, err)
	assert.True(t, finished)
	assert.False(t, passed)
	assert.Equal(t, "failed", status)
	finished, _, _, err = localJobOutcome(context.Background(), client, DefaultNamespace, "retrying")
	require.NoError(t, err)
	assert.False(t, finished)
}

func TestBreakpointShellInputStopsAtExplicitExit(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("pwd\nexit\ncontinue\n"))
	shell := &breakpointShellInput{reader: reader}
	contents, err := io.ReadAll(shell)
	require.NoError(t, err)
	assert.Equal(t, "pwd\nexit\n", string(contents))
	line, err := reader.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "continue\n", line)
}

func TestReadBreakpointLineReturnsPromptlyWhenContextIsCancelled(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	line, err := readBreakpointLine(ctx, bufio.NewReader(reader))
	assert.Empty(t, line)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPromptBreakpointReturnsAbortRequest(t *testing.T) {
	err := promptBreakpoint(context.Background(), nil, "local-abort", Options{In: strings.NewReader("a\n")}, io.Discard, io.Discard)
	assert.ErrorIs(t, err, errBreakpointAbort)
}

func TestFollowWorkflowAbortsWhenContextIsCancelled(t *testing.T) {
	controller := gomock.NewController(t)
	worker := executionworkertypes.NewMockWorker(controller)
	watcher := executionworkertypes.NewNotificationsWatcher()
	worker.EXPECT().Notifications(gomock.Any(), "local-follow", gomock.Any()).Return(watcher)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	abortCalls := 0

	result, err := followWorkflow(
		ctx,
		fake.NewClientset(),
		worker,
		&executionworkertypes.ExecuteResult{ScheduledAt: time.Now()},
		&PreparedRun{RunID: "local-follow", Namespace: DefaultNamespace},
		nil,
		func() error { abortCalls++; return nil },
		Options{},
		&bytes.Buffer{},
		&bytes.Buffer{},
	)
	assert.Nil(t, result)
	assert.True(t, IsInterruptedError(err), "expected interruption error, got %v", err)
	assert.Equal(t, 1, abortCalls)
}

func TestFollowWorkflowRendersStructuredLogBeforePassedResult(t *testing.T) {
	controller := gomock.NewController(t)
	worker := executionworkertypes.NewMockWorker(controller)
	watcher := executionworkertypes.NewNotificationsWatcher()
	worker.EXPECT().Notifications(gomock.Any(), "local-follow", gomock.Any()).Return(watcher)
	passed := testkube.PASSED_TestWorkflowStatus
	go func() {
		watcher.Send(&testkube.TestWorkflowExecutionNotification{EventType: "log", Log: "2026-09-03T07:00:00Z structured output\n"})
		watcher.Send(&testkube.TestWorkflowExecutionNotification{EventType: "result", Result: &testkube.TestWorkflowResult{Status: &passed, FinishedAt: time.Now()}})
		watcher.Close(nil)
	}()
	var output bytes.Buffer

	result, err := followWorkflow(
		context.Background(),
		fake.NewClientset(),
		worker,
		&executionworkertypes.ExecuteResult{ScheduledAt: time.Now()},
		&PreparedRun{RunID: "local-follow", Namespace: DefaultNamespace},
		nil,
		func() error { return nil },
		Options{},
		&output,
		io.Discard,
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Equal(t, "passed", result.Status)
	assert.Equal(t, "structured output\nresult: passed\n", output.String())
}

func TestFollowWorkflowDrainsTrailingStructuredLogsAfterJobFallback(t *testing.T) {
	controller := gomock.NewController(t)
	worker := executionworkertypes.NewMockWorker(controller)
	watcher := executionworkertypes.NewNotificationsWatcher()
	worker.EXPECT().Notifications(gomock.Any(), "local-follow", gomock.Any()).Return(watcher)
	client := fake.NewClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "local-follow", Namespace: DefaultNamespace},
		Status: batchv1.JobStatus{
			Succeeded: 1,
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
	})
	go func() {
		// Let the Job-status polling fallback observe completion first, then
		// deliver the log that would previously have been dropped on return.
		time.Sleep(localJobStatusPollInterval + 100*time.Millisecond)
		watcher.Send(&testkube.TestWorkflowExecutionNotification{EventType: "log", Log: "2026-09-03T07:00:00Z trailing output\n"})
		watcher.Close(nil)
	}()
	var output bytes.Buffer

	result, err := followWorkflow(
		context.Background(),
		client,
		worker,
		&executionworkertypes.ExecuteResult{ScheduledAt: time.Now()},
		&PreparedRun{RunID: "local-follow", Namespace: DefaultNamespace},
		nil,
		func() error { return nil },
		Options{},
		&output,
		io.Discard,
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Equal(t, "trailing output\nresult: passed\n", output.String())
}

func TestFollowWorkflowFallsBackToJobWhenNotificationsClose(t *testing.T) {
	controller := gomock.NewController(t)
	worker := executionworkertypes.NewMockWorker(controller)
	watcher := executionworkertypes.NewNotificationsWatcher()
	worker.EXPECT().Notifications(gomock.Any(), "local-follow", gomock.Any()).Return(watcher)
	watcher.Close(errors.New("watch disconnected"))
	client := fake.NewClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "local-follow", Namespace: DefaultNamespace},
		Status: batchv1.JobStatus{
			Succeeded: 1,
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
	})
	var output bytes.Buffer
	var errOutput bytes.Buffer

	result, err := followWorkflow(
		context.Background(),
		client,
		worker,
		&executionworkertypes.ExecuteResult{ScheduledAt: time.Now()},
		&PreparedRun{RunID: "local-follow", Namespace: DefaultNamespace},
		nil,
		func() error { return nil },
		Options{},
		&output,
		&errOutput,
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Equal(t, "result: passed\n", output.String())
	assert.Contains(t, errOutput.String(), "using Kubernetes Job status")
}
