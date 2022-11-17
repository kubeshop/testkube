package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestPodHasError(t *testing.T) {

	t.Run("succeded pod return no error ", func(t *testing.T) {
		// given
		pod := succeededPod()

		// when
		err := isPodFailed(pod)

		//then
		assert.NoError(t, err)
	})

	t.Run("failed pod returns error", func(t *testing.T) {
		// given
		pod := failedPod()

		// when
		err := isPodFailed(pod)

		//then
		assert.EqualError(t, err, "pod failed")
	})

	t.Run("failed pod with pending init container", func(t *testing.T) {
		// given
		pod := failedInitContainer()

		// when
		err := isPodFailed(pod)

		//then
		assert.EqualError(t, err, "secret nonexistingsecret not found")
	})
}

func succeededPod() *corev1.Pod {
	return &corev1.Pod{
		Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
	}
}

func failedPod() *corev1.Pod {
	return &corev1.Pod{
		Status: corev1.PodStatus{Phase: corev1.PodFailed, Message: "pod failed"},
	}
}

func failedInitContainer() *corev1.Pod {
	return &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "CreateContainerConfigError",
							Message: "secret nonexistingsecret not found",
						},
					},
				},
			}},
	}
}
