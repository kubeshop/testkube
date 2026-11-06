package kubernetesworker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func TestWorker_AbortInBackground(t *testing.T) {
	previousDelay, previousTimeout := abortRetryDelay, abortTimeout
	abortRetryDelay, abortTimeout = time.Millisecond, 50*time.Millisecond
	t.Cleanup(func() { abortRetryDelay, abortTimeout = previousDelay, previousTimeout })

	notFound := apierrors.NewNotFound(schema.GroupResource{Group: "batch", Resource: "jobs"}, "exec-1")

	unavailable := errors.New("the API server is not available")

	tests := []struct {
		name   string
		errors []error
		// wantCalls is the number of attempts; zero means more than one attempt
		wantCalls int32
	}{
		{name: "aborts one time when the first attempt works", errors: []error{nil}, wantCalls: 1},
		{name: "retries a transient failure", errors: []error{unavailable, nil}, wantCalls: 2},
		{name: "stops when the job is already gone, also behind the wrap of Abort", errors: []error{pkgerrors.Wrapf(notFound, "failed to patch job")}, wantCalls: 1},
		{name: "stops when the registry does not know the execution", errors: []error{registry.ErrResourceNotFound}, wantCalls: 1},
		{name: "stops when the time limit of the abort ends", errors: []error{unavailable}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &worker{}
			var calls atomic.Int32
			release := make(chan struct{})
			abort := func(context.Context) error {
				call := calls.Add(1)
				if call == 1 {
					// The first attempt waits, so the second request of the test comes while the abort runs.
					<-release
				}
				return tt.errors[min(int(call), len(tt.errors))-1]
			}

			done := w.abortInBackground("exec-1", abort)
			require.NotNil(t, done)
			assert.Nil(t, w.abortInBackground("exec-1", abort), "a second watch of the same execution must not start another abort")
			close(release)

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("the abort did not end")
			}
			if tt.wantCalls == 0 {
				assert.Greater(t, calls.Load(), int32(1))
			} else {
				assert.Equal(t, tt.wantCalls, calls.Load())
			}
		})
	}
}

// TestWorker_BuildInternalConfig_TLSRetry guards that the kubelet TLS log retry of the runner reaches
// the execution's internal config, which the toolkit reads to build the workers of parallel steps.
func TestWorker_BuildInternalConfig_TLSRetry(t *testing.T) {
	tests := []struct {
		name     string
		tlsRetry controller.TLSRetryConfig
		want     testworkflowconfig.WorkerConfig
	}{
		{
			name:     "carries the configured retry",
			tlsRetry: controller.TLSRetryConfig{MaxAttempts: 12, InitialDelay: 2 * time.Second, MaxDelay: time.Minute},
			want: testworkflowconfig.WorkerConfig{
				LogsTLSRetryMaxAttempts:  12,
				LogsTLSRetryInitialDelay: 2 * time.Second,
				LogsTLSRetryMaxDelay:     time.Minute,
			},
		},
		{
			// Zero values are omitted from the serialized config, so the controller defaults apply.
			name:     "leaves the retry empty when none is configured",
			tlsRetry: controller.TLSRetryConfig{},
			want:     testworkflowconfig.WorkerConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWorker(fake.NewSimpleClientset(), nil, Config{
				Cluster:  ClusterConfig{DefaultNamespace: "ns", Namespaces: map[string]NamespaceConfig{"ns": {}}},
				TLSRetry: tt.tlsRetry,
			})

			cfg := w.buildInternalConfig("exec-1", "", testworkflowconfig.ExecutionConfig{Id: "exec-1"}, testworkflowconfig.ControlPlaneConfig{}, testworkflowsv1.TestWorkflow{}, "")

			assert.Equal(t, tt.want.LogsTLSRetryMaxAttempts, cfg.Worker.LogsTLSRetryMaxAttempts)
			assert.Equal(t, tt.want.LogsTLSRetryInitialDelay, cfg.Worker.LogsTLSRetryInitialDelay)
			assert.Equal(t, tt.want.LogsTLSRetryMaxDelay, cfg.Worker.LogsTLSRetryMaxDelay)
		})
	}
}
