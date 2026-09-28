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

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
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
