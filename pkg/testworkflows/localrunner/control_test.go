package localrunner

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalControlActivePodSelectsOnlyWorkflowComponent(t *testing.T) {
	labels, err := Labels("local-control", "workflow")
	require.NoError(t, err)
	relayLabels, err := Labels("local-control", "source-relay")
	require.NoError(t, err)
	client := fake.NewClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "relay", Namespace: DefaultNamespace, Labels: relayLabels}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "workflow", Namespace: DefaultNamespace, Labels: labels}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
	)
	control := NewLocalControl(client, nil, DefaultNamespace)
	pod, err := control.ActivePod(context.Background(), "local-control")
	require.NoError(t, err)
	assert.Equal(t, "workflow", pod.Name)
}

func TestActiveContainerUsesRunningInitThenRegularContainer(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "workflow"}, Status: corev1.PodStatus{
		InitContainerStatuses: []corev1.ContainerStatus{{Name: "init", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}},
		ContainerStatuses:     []corev1.ContainerStatus{{Name: "final", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}},
	}}
	container, err := activeContainer(pod)
	require.NoError(t, err)
	assert.Equal(t, "init", container)

	pod.Status.InitContainerStatuses[0].State.Running = nil
	container, err = activeContainer(pod)
	require.NoError(t, err)
	assert.Equal(t, "final", container)
}

func TestLocalControlActivePodRejectsAmbiguousOrTerminalPods(t *testing.T) {
	labels, err := Labels("local-control", "workflow")
	require.NoError(t, err)
	client := fake.NewClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "first", Namespace: DefaultNamespace, Labels: labels}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "second", Namespace: DefaultNamespace, Labels: labels}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
	)
	control := NewLocalControl(client, nil, DefaultNamespace)
	_, err = control.ActivePod(context.Background(), "local-control")
	require.Error(t, err)
	assert.ErrorContains(t, err, "multiple active workflow pods")
}

func TestLocalControlPodDetailsIncludesOnlyExactPodEventsNewestFirst(t *testing.T) {
	labels, err := Labels("local-control", "workflow")
	require.NoError(t, err)
	old := metav1.NewTime(time.Now().Add(-time.Minute))
	newer := metav1.NewTime(time.Now())
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "workflow", Namespace: DefaultNamespace, UID: "workflow-uid", Labels: labels}, Status: corev1.PodStatus{Phase: corev1.PodRunning}}
	client := fake.NewClientset(
		pod,
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: DefaultNamespace}, InvolvedObject: corev1.ObjectReference{Name: "workflow", UID: "workflow-uid"}, Reason: "Old", LastTimestamp: old},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: DefaultNamespace}, InvolvedObject: corev1.ObjectReference{Name: "workflow", UID: "workflow-uid"}, Reason: "New", LastTimestamp: newer},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: DefaultNamespace}, InvolvedObject: corev1.ObjectReference{Name: "source-relay", UID: "relay-uid"}, Reason: "Ignore", LastTimestamp: newer},
	)
	control := NewLocalControl(client, nil, DefaultNamespace)

	details, events, err := control.PodDetails(context.Background(), "local-control")
	require.NoError(t, err)
	assert.Equal(t, "workflow", details.Name)
	require.Len(t, events, 2)
	assert.Equal(t, "New", events[0].Reason)
	assert.Equal(t, "Old", events[1].Reason)
}
