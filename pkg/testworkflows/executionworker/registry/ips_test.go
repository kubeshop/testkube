package registry_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestPodIpsRegistryGet(t *testing.T) {
	for _, tc := range []struct {
		name   string
		hasPod bool
		podIP  string
		err    error
	}{
		{name: "missing pod", err: registry.ErrResourceNotFound},
		{name: "pod without IP", hasPod: true},
		{name: "pod with IP", hasPod: true, podIP: "192.0.2.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientset()
			if tc.hasPod {
				_, err := client.CoreV1().Pods("test").Create(t.Context(), &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "execution-pod",
						Labels: map[string]string{constants.ResourceIdLabelName: "execution"},
					},
					Status: corev1.PodStatus{PodIP: tc.podIP},
				}, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			ips := registry.NewPodIpsRegistry(client, func(context.Context, string) (string, error) {
				return "test", nil
			}, 10)

			ip, err := ips.Get(t.Context(), "execution")

			require.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.podIP, ip)
		})
	}
}

// A missing pod must not prevent a later lookup from finding the execution's pod.
func TestPodIpsRegistryGetMissingPodNotCached(t *testing.T) {
	client := fake.NewClientset()
	ips := registry.NewPodIpsRegistry(client, func(context.Context, string) (string, error) {
		return "test", nil
	}, 10)

	ip, err := ips.Get(t.Context(), "execution")
	require.ErrorIs(t, err, registry.ErrResourceNotFound)
	assert.Empty(t, ip)

	_, err = client.CoreV1().Pods("test").Create(t.Context(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "execution-pod",
			Labels: map[string]string{constants.ResourceIdLabelName: "execution"},
		},
		Status: corev1.PodStatus{PodIP: "192.0.2.1"},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	ip, err = ips.Get(t.Context(), "execution")
	require.NoError(t, err)
	assert.Equal(t, "192.0.2.1", ip)
}
