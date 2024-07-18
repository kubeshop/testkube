package testworkflowcontroller

import (
	"regexp"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	KubernetesLogTimeFormat         = "2006-01-02T15:04:05.000000000Z"
	KubernetesTimezoneLogTimeFormat = KubernetesLogTimeFormat + "07:00"
)

func GetEventContainerName(event *corev1.Event) string {
	regex := regexp.MustCompile(`^spec\.(?:initContainers|containers)\{([^]]+)}`)
	path := event.InvolvedObject.FieldPath
	if regex.Match([]byte(path)) {
		name := regex.ReplaceAllString(event.InvolvedObject.FieldPath, "$1")
		return name
	}
	return ""
}

func IsPodDone(pod *corev1.Pod) bool {
	return (pod.Status.Phase != corev1.PodPending && pod.Status.Phase != corev1.PodRunning) || pod.ObjectMeta.DeletionTimestamp != nil
}

func IsJobDone(job *batchv1.Job) bool {
	return (job.Status.Active == 0 && (job.Status.Succeeded > 0 || job.Status.Failed > 0)) || job.ObjectMeta.DeletionTimestamp != nil
}

type ContainerResultStep struct {
	Status     testkube.TestWorkflowStepStatus
	ExitCode   int
	Details    string
	FinishedAt time.Time
}

type ContainerResult struct {
	Steps      []ContainerResultStep
	Details    string
	ExitCode   int
	FinishedAt time.Time
}

var UnknownContainerResult = ContainerResult{
	ExitCode: -1,
}
