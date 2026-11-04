package registry

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestKubeReadPolicy_RetryIf(t *testing.T) {
	pods := schema.GroupResource{Resource: "pods"}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "a transient error is retried", err: stderrors.New("kube-apiserver unavailable"), want: true},
		{name: "a missing object is not retried", err: k8serrors.NewNotFound(pods, "x"), want: false},
		{name: "a forbidden read is not retried", err: k8serrors.NewForbidden(pods, "x", stderrors.New("nope")), want: false},
		{name: "an unauthorized read is not retried", err: k8serrors.NewUnauthorized("no token"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, kubeReadPolicy.RetryIf(tt.err))
		})
	}
}
