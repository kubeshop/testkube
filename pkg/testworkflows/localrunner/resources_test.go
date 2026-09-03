package localrunner

import (
	"context"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceManagerCleanSelectsOnlyExactRunLabels(t *testing.T) {
	ctx := context.Background()
	labelsA, err := Labels("local-demo-a", "workflow")
	require.NoError(t, err)
	labelsB, err := Labels("local-demo-b", "workflow")
	require.NoError(t, err)
	client := fake.NewClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: DefaultNamespace}},
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "job-a", Namespace: DefaultNamespace, Labels: labelsA}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: DefaultNamespace, Labels: labelsA}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service-a", Namespace: DefaultNamespace, Labels: labelsA}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret-a", Namespace: DefaultNamespace, Labels: labelsA}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "config-a", Namespace: DefaultNamespace, Labels: labelsA}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "pvc-a", Namespace: DefaultNamespace, Labels: labelsA}},
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "job-b", Namespace: DefaultNamespace, Labels: labelsB}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "user-secret", Namespace: DefaultNamespace}},
	)

	manager := NewResourceManager(client, DefaultNamespace)
	require.NoError(t, manager.Clean(ctx, "local-demo-a"))

	_, err = client.BatchV1().Jobs(DefaultNamespace).Get(ctx, "job-a", metav1.GetOptions{})
	assert.Error(t, err)
	_, err = client.CoreV1().Pods(DefaultNamespace).Get(ctx, "pod-a", metav1.GetOptions{})
	assert.Error(t, err)
	_, err = client.CoreV1().Secrets(DefaultNamespace).Get(ctx, "secret-a", metav1.GetOptions{})
	assert.Error(t, err)

	_, err = client.BatchV1().Jobs(DefaultNamespace).Get(ctx, "job-b", metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().Secrets(DefaultNamespace).Get(ctx, "user-secret", metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().Namespaces().Get(ctx, DefaultNamespace, metav1.GetOptions{})
	require.NoError(t, err)

	// A second exact cleanup is deliberately harmless.
	require.NoError(t, manager.Clean(ctx, "local-demo-a"))
}

func TestResourceManagerEnsureNamespaceAndReportsStaleRuns(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	manager := NewResourceManager(client, DefaultNamespace)
	require.NoError(t, manager.EnsureNamespace(ctx))
	namespace, err := client.CoreV1().Namespaces().Get(ctx, DefaultNamespace, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, localNamespaceLabelValue, namespace.Labels[LocalLabel])

	labels, err := Labels("local-old", "workflow")
	require.NoError(t, err)
	old := time.Now().Add(-staleRunThreshold - time.Minute)
	client.Tracker().Add(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: DefaultNamespace, Labels: labels, CreationTimestamp: metav1.NewTime(old)}})
	newLabels, err := Labels("local-new", "workflow")
	require.NoError(t, err)
	client.Tracker().Add(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: DefaultNamespace, Labels: newLabels, CreationTimestamp: metav1.NewTime(time.Now())}})
	// A Job has a one-hour TTL. Verify a left-over relay resource still causes
	// a warning after its Job has already been removed by that fallback TTL.
	relayLabels, err := Labels("local-relay-leftover", "source-relay")
	require.NoError(t, err)
	client.Tracker().Add(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "old-relay", Namespace: DefaultNamespace, Labels: relayLabels, CreationTimestamp: metav1.NewTime(old)}})

	stale, err := manager.StaleRunIDs(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, []string{"local-old", "local-relay-leftover"}, stale)
}
