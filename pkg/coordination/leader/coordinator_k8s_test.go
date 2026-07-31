package leader

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	k8sbackend "github.com/kubeshop/testkube/pkg/repository/leasebackend/k8s"
)

// Going through the real lease backend rather than a scripted one is what proves a throttled API
// server arrives as an error from the lease call, not as a lease held by someone else. That
// distinction is the whole point of the tolerance window and only exists once both halves are wired.
func TestCoordinator_evaluateWithK8sBackend(t *testing.T) {
	const leaseDuration = time.Minute

	tests := []struct {
		name string
		// throttleFor is how much time passes while every lease call is rejected.
		throttleFor time.Duration
		// recovers ends the rejections before the last evaluation.
		recovers   bool
		wantStops  int32
		wantStarts int32
	}{
		{
			name:        "keeps the tasks running while throttling stays inside the lease duration",
			throttleFor: leaseDuration / 2,
			wantStops:   0,
			wantStarts:  1,
		},
		{
			name:        "stops the tasks once throttling outlasts the lease duration",
			throttleFor: leaseDuration + time.Second,
			wantStops:   1,
			wantStarts:  1,
		},
		{
			name:        "never restarts the tasks when throttling ends inside the lease duration",
			throttleFor: leaseDuration / 2,
			recovers:    true,
			wantStops:   0,
			wantStarts:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var throttled atomic.Bool
			clientset := fake.NewClientset()
			clientset.PrependReactor("*", "leases", func(ktesting.Action) (bool, runtime.Object, error) {
				if !throttled.Load() {
					return false, nil, nil
				}

				return true, nil, apierrors.NewTooManyRequestsError("the server has received too many requests")
			})

			clock := &fakeClock{current: time.Now()}
			task := &countingTask{}
			coord := New(k8sbackend.NewK8sLeaseBackend(clientset, "testkube", "testkube"), "id-1", "cluster-1", nil,
				WithLeaseDuration(leaseDuration),
				withClock(clock.now))
			coord.Register(task.task())

			coord.evaluate(ctx)
			require.Eventually(t, func() bool {
				return task.starts.Load() == 1
			}, waitTimeout, 5*time.Millisecond, "the coordinator should lead while the api server is healthy")

			throttled.Store(true)
			clock.advance(tt.throttleFor)
			coord.evaluate(ctx)

			if tt.recovers {
				throttled.Store(false)
				coord.evaluate(ctx)
			}

			require.Eventually(t, func() bool {
				return task.stops.Load() == tt.wantStops
			}, waitTimeout, 5*time.Millisecond, "expected %d task stops", tt.wantStops)
			assert.Equal(t, tt.wantStarts, task.starts.Load())
		})
	}
}
