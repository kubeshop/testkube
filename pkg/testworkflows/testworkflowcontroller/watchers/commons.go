package watchers

import (
	"context"
	"errors"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	defaultListTimeoutSeconds  = int64(30)
	defaultWatchTimeoutSeconds = int64(365 * 24 * 3600)
)

var (
	ErrDone = errors.New("resource is done")
)

type kubernetesClient[T any, U any] interface {
	List(ctx context.Context, options metav1.ListOptions) (*T, error)
	Watch(ctx context.Context, options metav1.ListOptions) (watch.Interface, error)
}

func IsJobFinished(job *batchv1.Job) bool {
	if job == nil {
		return true
	}
	if job.ObjectMeta.DeletionTimestamp != nil {
		return true
	}
	if job.Status.CompletionTime != nil {
		return true
	}
	if job.Status.Active == 0 && (job.Status.Succeeded > 0 && job.Status.Failed > 0) {
		return true
	}
	for i := range job.Status.Conditions {
		if job.Status.Conditions[i].Type == batchv1.JobComplete || job.Status.Conditions[i].Type == batchv1.JobFailed {
			if job.Status.Conditions[i].Status == corev1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func IsPodFinished(pod *corev1.Pod) bool {
	if pod == nil {
		return true
	}
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return true
	}
	if pod.ObjectMeta.DeletionTimestamp != nil {
		return true
	}
	if pod.Status.Phase == corev1.PodUnknown {
		for i := range pod.Status.Conditions {
			if pod.Status.Conditions[i].Type == corev1.DisruptionTarget && pod.Status.Conditions[i].Status == corev1.ConditionTrue {
				return true
			}
		}
	}
	return false
}
