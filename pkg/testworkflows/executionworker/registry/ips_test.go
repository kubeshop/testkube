package registry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestIpsRegistry_Get(t *testing.T) {
	const namespace = "testkube"
	const id = "execution-1"

	pod := func(ip string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      id,
				Namespace: namespace,
				Labels:    map[string]string{constants.ResourceIdLabelName: id},
			},
			Status: corev1.PodStatus{PodIP: ip},
		}
	}

	tests := []struct {
		name    string
		objects []runtime.Object
		wantIP  string
		wantErr error
	}{
		{
			name:    "returns the ip of the pod",
			objects: []runtime.Object{pod("10.0.0.1")},
			wantIP:  "10.0.0.1",
		},
		{
			name:    "returns an empty ip while the pod has none yet",
			objects: []runtime.Object{pod("")},
			wantIP:  "",
		},
		{
			name:    "reports a missing resource when the pod is gone",
			objects: nil,
			wantErr: ErrResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewPodIpsRegistry(fake.NewSimpleClientset(tt.objects...), func(ctx context.Context, id string) (string, error) {
				return namespace, nil
			}, 10)

			ip, err := registry.Get(context.Background(), id)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantIP, ip)
		})
	}
}
