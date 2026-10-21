package registry

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestKubeReadPolicy(t *testing.T) {
	transient := stderrors.New("kube-apiserver unavailable")
	pods := schema.GroupResource{Resource: "pods"}
	tests := []struct {
		name         string
		err          error
		failures     int
		wantAttempts int
	}{
		{name: "a transient error is retried until the read works", err: transient, failures: 1, wantAttempts: 2},
		{name: "a transient error that stays uses every attempt", err: transient, failures: 10, wantAttempts: kubeReadPolicy.Attempts},
		{name: "a missing object is not retried", err: k8serrors.NewNotFound(pods, "x"), failures: 10, wantAttempts: 1},
		{name: "a forbidden read is not retried", err: k8serrors.NewForbidden(pods, "x", stderrors.New("nope")), failures: 10, wantAttempts: 1},
		{name: "an unauthorized read is not retried", err: k8serrors.NewUnauthorized("no token"), failures: 10, wantAttempts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := kubeReadPolicy
			policy.Delay = time.Millisecond
			calls := 0
			err := policy.Do(context.Background(), func() error {
				calls++
				if calls <= tt.failures {
					return tt.err
				}
				return nil
			})

			assert.Equal(t, tt.wantAttempts, calls)
			if tt.failures < tt.wantAttempts {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.err)
		})
	}
}
