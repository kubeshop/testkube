package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	secretenv "github.com/kubeshop/testkube/pkg/executor/secret"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	// GitUsernameSecretName is git username secret name
	GitUsernameSecretName = "git-username"
	// GitUsernameEnvVarName is git username environment var name
	GitUsernameEnvVarName = "RUNNER_GITUSERNAME"
	// GitTokenSecretName is git token secret name
	GitTokenSecretName = "git-token"
	// GitTokenEnvVarName is git token environment var name
	GitTokenEnvVarName = "RUNNER_GITTOKEN"
	pollTimeout        = 24 * time.Hour
	pollInterval       = 200 * time.Millisecond
)

var RunnerEnvVars = []corev1.EnvVar{
	{
		Name:  "DEBUG",
		Value: os.Getenv("DEBUG"),
	},
	{
		Name:  "RUNNER_ENDPOINT",
		Value: os.Getenv("STORAGE_ENDPOINT"),
	},
	{
		Name:  "RUNNER_ACCESSKEYID",
		Value: os.Getenv("STORAGE_ACCESSKEYID"),
	},
	{
		Name:  "RUNNER_SECRETACCESSKEY",
		Value: os.Getenv("STORAGE_SECRETACCESSKEY"),
	},
	{
		Name:  "RUNNER_LOCATION",
		Value: os.Getenv("STORAGE_LOCATION"),
	},
	{
		Name:  "RUNNER_TOKEN",
		Value: os.Getenv("STORAGE_TOKEN"),
	},
	{
		Name:  "RUNNER_SSL",
		Value: os.Getenv("STORAGE_SSL"),
	},
	{
		Name:  "RUNNER_SCRAPPERENABLED",
		Value: os.Getenv("SCRAPPERENABLED"),
	},
	{
		Name:  "RUNNER_DATADIR",
		Value: "/data",
	},
}

// IsPodReady defines if pod is ready or failed for logs scrapping
func IsPodReady(c kubernetes.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			return true, nil
		}

		if err = isPodFailed(pod); err != nil {
			return true, err
		}

		return false, nil
	}
}

// isWaitStateFailed defines possible failed wait state
// those states are defined and throwed as errors in Kubernetes runtime
// https://github.com/kubernetes/kubernetes/blob/127f33f63d118d8d61bebaba2a240c60f71c824a/pkg/kubelet/kuberuntime/kuberuntime_container.go#L59
func isWaitStateFailed(state string) bool {
	var failedWaitingStates = []string{
		"CreateContainerConfigError",
		"PreCreateHookError",
		"CreateContainerError",
		"PreStartHookError",
		"PostStartHookError",
	}

	for _, fws := range failedWaitingStates {
		if state == fws {
			return true
		}
	}

	return false
}

// pod can be in wait state with reason which is error for us on the end
func isPodFailed(pod *corev1.Pod) (err error) {
	if pod.Status.Phase == corev1.PodFailed {
		return errors.New(pod.Status.Message)
	}

	for _, initContainerStatus := range pod.Status.InitContainerStatuses {
		waitState := initContainerStatus.State.Waiting
		// TODO there could be more edge cases but didn't found any constants in go libraries
		if waitState != nil && isWaitStateFailed(waitState.Reason) {
			return errors.New(waitState.Message)
		}
	}

	return
}

// GetJobPods returns job pods
func GetJobPods(podsClient tcorev1.PodInterface, jobName string, retryNr, retryCount int) (*corev1.PodList, error) {
	pods, err := podsClient.List(context.Background(), metav1.ListOptions{LabelSelector: "job-name=" + jobName})
	if err != nil {
		return nil, err
	}
	if retryNr == retryCount {
		return nil, fmt.Errorf("retry count exceeeded, there are no active pods with given id=%s", jobName)
	}
	if len(pods.Items) == 0 {
		time.Sleep(time.Duration(retryNr * 500 * int(time.Millisecond))) // increase backoff timeout
		return GetJobPods(podsClient, jobName, retryNr+1, retryCount)
	}
	return pods, nil
}

// GetPodLogs returns pod logs bytes
func GetPodLogs(c kubernetes.Interface, namespace string, pod corev1.Pod, logLinesCount ...int64) (logs []byte, err error) {
	count := int64(100)
	if len(logLinesCount) > 0 {
		count = logLinesCount[0]
	}

	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	for _, container := range containers {
		podLogOptions := corev1.PodLogOptions{
			Follow:    false,
			TailLines: &count,
			Container: container,
		}

		podLogRequest := c.CoreV1().
			Pods(namespace).
			GetLogs(pod.Name, &podLogOptions)

		stream, err := podLogRequest.Stream(context.Background())
		if err != nil {
			if len(logs) != 0 && strings.Contains(err.Error(), "PodInitializing") {
				return logs, nil
			}

			return logs, err
		}

		defer stream.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, stream)
		if err != nil {
			if len(logs) != 0 && strings.Contains(err.Error(), "PodInitializing") {
				return logs, nil
			}

			return logs, err
		}

		logs = append(logs, buf.Bytes()...)
	}

	return logs, nil
}

// AbortJob - aborts Kubernetes Job with no grace period
func AbortJob(c kubernetes.Interface, namespace string, jobName string) *testkube.ExecutionResult {
	var zero int64 = 0
	bg := metav1.DeletePropagationBackground
	jobs := c.BatchV1().Jobs(namespace)
	err := jobs.Delete(context.TODO(), jobName, metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
		PropagationPolicy:  &bg,
	})
	if err != nil {
		return &testkube.ExecutionResult{
			Status: testkube.ExecutionStatusFailed,
			Output: err.Error(),
		}
	}
	return &testkube.ExecutionResult{
		Status: testkube.ExecutionStatusCancelled,
	}
}

// PrepareSecretEnvs prepares all the secrets for templating
func PrepareSecretEnvs(secretEnvs map[string]string, variables map[string]testkube.Variable,
	usernameSecret, tokenSecret *testkube.SecretRef) []corev1.EnvVar {

	secretEnvVars := secretenv.NewEnvManager().Prepare(secretEnvs, variables)

	// prepare git credentials
	var data = []struct {
		envVar    string
		secretRef *testkube.SecretRef
	}{
		{
			GitUsernameEnvVarName,
			usernameSecret,
		},
		{
			GitTokenEnvVarName,
			tokenSecret,
		},
	}

	for _, value := range data {
		if value.secretRef != nil {
			secretEnvVars = append(secretEnvVars, corev1.EnvVar{
				Name: value.envVar,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: value.secretRef.Name,
						},
						Key: value.secretRef.Key,
					},
				},
			})
		}
	}

	return secretEnvVars
}
