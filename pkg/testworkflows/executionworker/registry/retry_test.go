package registry

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestKubeReadRetry_TransientErrorRetriedThenSucceeds(t *testing.T) {
	var calls int
	transient := stderrors.New("kube-apiserver unavailable")
	err := kubeReadRetry(func() error {
		calls++
		if calls < 2 {
			return transient
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestKubeReadRetry_GivesUpAfterBudget(t *testing.T) {
	var calls int
	transient := stderrors.New("kube-apiserver unavailable")
	err := kubeReadRetry(func() error {
		calls++
		return transient
	})
	assert.ErrorIs(t, err, transient)
	assert.Equal(t, kubeReadRetryCount, calls)
}

func TestKubeReadRetry_NotFoundShortCircuits(t *testing.T) {
	var calls int
	nf := k8serrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "x")
	err := kubeReadRetry(func() error {
		calls++
		return nf
	})
	assert.True(t, k8serrors.IsNotFound(err))
	assert.Equal(t, 1, calls)
}

func TestKubeReadRetry_ForbiddenShortCircuits(t *testing.T) {
	var calls int
	f := k8serrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "x", stderrors.New("nope"))
	err := kubeReadRetry(func() error {
		calls++
		return f
	})
	assert.True(t, k8serrors.IsForbidden(err))
	assert.Equal(t, 1, calls)
}
