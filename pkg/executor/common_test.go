package executor

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/kubernetes/fake"
)

func TestPodHasError(t *testing.T) {

	t.Run("succeded pod return no error ", func(t *testing.T) {
		// given
		pod := succeededPod()

		// when
		err := IsPodFailed(pod)

		//then
		assert.NoError(t, err)
	})

	t.Run("failed pod returns error", func(t *testing.T) {
		// given
		pod := failedPod()

		// when
		err := IsPodFailed(pod)

		//then
		assert.EqualError(t, err, "pod failed")
	})

	t.Run("failed pod with pending init container", func(t *testing.T) {
		// given
		pod := failedInitContainer()

		// when
		err := IsPodFailed(pod)

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

func TestGetPodLogs(t *testing.T) {
	type args struct {
		c             kubernetes.Interface
		namespace     string
		pod           corev1.Pod
		logLinesCount []int64
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []byte
		wantErr  bool
	}{
		{
			name: "pod with multiple containers",
			args: args{
				c:         fake.NewSimpleClientset(),
				namespace: "testkube_test",
				pod: corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "1234-init",
							},
						},
						Containers: []corev1.Container{
							{
								Name: "1234",
							},
							{
								Name: "second_container",
							},
						},
					},
				},
			},
			wantLogs: []byte("fake logsfake logs"),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whitelistedContainers := []string{"logs", "init", "scraper"}
			gotLogs, err := GetPodLogs(context.Background(), tt.args.c, tt.args.namespace, tt.args.pod, "1234", whitelistedContainers, tt.args.logLinesCount...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPodLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotLogs, tt.wantLogs) {
				t.Errorf("GetPodLogs() = %v, want %v", gotLogs, tt.wantLogs)
			}
		})
	}
}

func TestIsWhitelistedContainer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		containerName string
		id            string
		expected      bool
	}{
		{"mycontainer", "mycontainer", true},
		{"mycontainer-init", "mycontainer", true},
		{"mycontainer-scraper", "mycontainer", true},
		{"mycontainer-logs", "mycontainer", true},
		{"anothercontainer", "mycontainer", false},
		{"istio-init", "mycontainer", false},
		{"istio-proxy", "mycontainer", false},
		{"scraper-mycontainer", "mycontainer", false},
		{"logs-mycontainer", "mycontainer", false},
		{"", "mycontainer", false},
		{"mycontainer", "", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("containerName: %s, id: %s", tt.containerName, tt.id), func(t *testing.T) {
			whitelisted := []string{"logs", "init", "scraper"}
			result := IsWhitelistedContainer(tt.containerName, tt.id, whitelisted)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
