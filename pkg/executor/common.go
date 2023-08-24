package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	executorsmapper "github.com/kubeshop/testkube/pkg/mapper/executors"
)

var ErrPodInitializing = errors.New("PodInitializing")

const (
	// VolumeDir is volume dir
	VolumeDir            = "/data"
	defaultLogLinesCount = 100
	// GitUsernameSecretName is git username secret name
	GitUsernameSecretName = "git-username"
	// GitTokenSecretName is git token secret name
	GitTokenSecretName = "git-token"
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
		Name:  "RUNNER_REGION",
		Value: os.Getenv("STORAGE_REGION"),
	},
	{
		Name:  "RUNNER_TOKEN",
		Value: os.Getenv("STORAGE_TOKEN"),
	},

	{
		Name:  "RUNNER_SSL",
		Value: getOr("STORAGE_SSL", "false"),
	},
	{
		Name:  "RUNNER_SCRAPPERENABLED",
		Value: getOr("SCRAPPERENABLED", "false"),
	},
	{
		Name:  "RUNNER_DATADIR",
		Value: VolumeDir,
	},
	{
		Name:  "RUNNER_CDEVENTS_TARGET",
		Value: os.Getenv("CDEVENTS_TARGET"),
	},
	{
		Name:  "RUNNER_COMPRESSARTIFACTS",
		Value: getOr("COMPRESSARTIFACTS", "false"),
	},
	{
		Name:  "RUNNER_CLOUD_MODE",
		Value: getRunnerCloudMode(),
	},
	{
		Name:  "RUNNER_CLOUD_API_KEY",
		Value: os.Getenv("TESTKUBE_CLOUD_API_KEY"),
	},
	{
		Name:  "RUNNER_CLOUD_API_TLS_INSECURE",
		Value: getOr("TESTKUBE_CLOUD_TLS_INSECURE", "false"),
	},
	{
		Name:  "RUNNER_CLOUD_API_URL",
		Value: os.Getenv("TESTKUBE_CLOUD_URL"),
	},
	{
		Name:  "RUNNER_DASHBOARD_URI",
		Value: os.Getenv("TESTKUBE_DASHBOARD_URI"),
	},
	{
		Name:  "CI",
		Value: "1",
	},
}

func getOr(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getRunnerCloudMode() string {
	val := "false"
	if os.Getenv("TESTKUBE_CLOUD_API_KEY") != "" {
		val = "true"
	}
	return val
}

// Templates contains templates for executor
type Templates struct {
	Job     string `json:"job"`
	PVC     string `json:"pvc"`
	Scraper string `json:"scraper"`
}

// Images contains images for executor
type Images struct {
	Init    string
	Scraper string
}

// IsPodReady defines if pod is ready or failed for logs scrapping
func IsPodReady(ctx context.Context, c kubernetes.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			return true, nil
		}

		if err = IsPodFailed(pod); err != nil {
			return true, err
		}

		return false, nil
	}
}

// IsPodLoggable defines if pod is ready to get logs from it
func IsPodLoggable(ctx context.Context, c kubernetes.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}

		if err = IsPodFailed(pod); err != nil {
			return true, err
		}

		return false, nil
	}
}

// IsWaitStateFailed defines possible failed wait state
// those states are defined and throwed as errors in Kubernetes runtime
// https://github.com/kubernetes/kubernetes/blob/127f33f63d118d8d61bebaba2a240c60f71c824a/pkg/kubelet/kuberuntime/kuberuntime_container.go#L59
func IsWaitStateFailed(state string) bool {
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

// IsPodFailed checks if pod failed
// pod can be in wait state with reason which is error for us on the end
func IsPodFailed(pod *corev1.Pod) (err error) {
	if pod.Status.Phase == corev1.PodFailed {
		return errors.New(pod.Status.Message)
	}

	for _, initContainerStatus := range pod.Status.InitContainerStatuses {
		waitState := initContainerStatus.State.Waiting
		// TODO there could be more edge cases but didn't found any constants in go libraries
		if waitState != nil && IsWaitStateFailed(waitState.Reason) {
			return errors.New(waitState.Message)
		}
	}

	return
}

// GetJobPods returns job pods
func GetJobPods(ctx context.Context, podsClient tcorev1.PodInterface, jobName string, retryNr, retryCount int) (*corev1.PodList, error) {
	pods, err := podsClient.List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + jobName})
	if err != nil {
		return nil, err
	}
	if retryNr == retryCount {
		return nil, fmt.Errorf("retry count exceeeded, there are no active pods with given id=%s", jobName)
	}
	if len(pods.Items) == 0 {
		time.Sleep(time.Duration(retryNr * 500 * int(time.Millisecond))) // increase backoff timeout
		return GetJobPods(ctx, podsClient, jobName, retryNr+1, retryCount)
	}
	return pods, nil
}

// GetPodLogs returns pod logs bytes
func GetPodLogs(ctx context.Context, c kubernetes.Interface, namespace string, pod corev1.Pod, logLinesCount ...int64) (logs []byte, err error) {
	var count int64 = defaultLogLinesCount
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
		containerLogs, err := GetContainerLogs(ctx, c, &pod, container, namespace, &count)
		if err != nil {
			if errors.Is(err, ErrPodInitializing) {
				return logs, nil
			}
			return logs, err
		}

		logs = append(logs, containerLogs...)
	}

	return logs, nil
}

// GetContainerLogs returns container logs
func GetContainerLogs(ctx context.Context, c kubernetes.Interface, pod *corev1.Pod, container, namespace string, tailLines *int64) ([]byte, error) {
	podLogOptions := corev1.PodLogOptions{
		Container: container,
	}

	podLogRequest := c.CoreV1().
		Pods(namespace).
		GetLogs(pod.Name, &podLogOptions)

	stream, err := podLogRequest.Stream(ctx)
	if err != nil {
		isPodInitializingError := strings.Contains(err.Error(), "PodInitializing")
		if isPodInitializingError {
			return nil, errors.WithStack(ErrPodInitializing)
		}

		return nil, err
	}
	defer stream.Close()

	var buff bytes.Buffer
	_, err = io.Copy(&buff, stream)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// AbortJob - aborts Kubernetes Job with no grace period
func AbortJob(ctx context.Context, c kubernetes.Interface, namespace string, jobName string) (*testkube.ExecutionResult, error) {
	var zero int64 = 0
	bg := metav1.DeletePropagationBackground
	jobs := c.BatchV1().Jobs(namespace)
	err := jobs.Delete(ctx, jobName, metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
		PropagationPolicy:  &bg,
	})
	if err != nil {
		log.DefaultLogger.Errorf("Error while aborting job %s: %s", jobName, err.Error())
		return &testkube.ExecutionResult{
			Status: testkube.ExecutionStatusFailed,
			Output: err.Error(),
		}, nil
	}
	log.DefaultLogger.Infof("Job %s aborted", jobName)
	return &testkube.ExecutionResult{
		Status: testkube.ExecutionStatusAborted,
	}, nil
}

// SyncDefaultExecutors creates or updates default executors
func SyncDefaultExecutors(
	executorsClient executorsclientv1.Interface,
	namespace string,
	executors []testkube.ExecutorDetails,
	readOnlyExecutors bool,
) (images Images, err error) {
	if len(executors) == 0 {
		return images, nil
	}

	for _, executor := range executors {
		if executor.Executor == nil {
			continue
		}

		if executor.Name == "init-executor" {
			images.Init = executor.Executor.Image
			continue
		}

		if executor.Name == "scraper-executor" {
			images.Scraper = executor.Executor.Image
			continue
		}

		if readOnlyExecutors {
			continue
		}

		obj := &executorv1.Executor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      executor.Name,
				Namespace: namespace,
			},
			Spec: executorv1.ExecutorSpec{
				Types:        executor.Executor.Types,
				ExecutorType: executorv1.ExecutorType(executor.Executor.ExecutorType),
				Image:        executor.Executor.Image,
				Command:      executor.Executor.Command,
				Args:         executor.Executor.Args,
				Features:     executorsmapper.MapFeaturesToCRD(executor.Executor.Features),
				ContentTypes: executorsmapper.MapContentTypesToCRD(executor.Executor.ContentTypes),
				Meta:         executorsmapper.MapMetaToCRD(executor.Executor.Meta),
			},
		}

		result, err := executorsClient.Get(executor.Name)
		if err != nil && !k8serrors.IsNotFound(err) {
			return images, err
		}
		if err != nil {
			if _, err = executorsClient.Create(obj); err != nil {
				return images, err
			}
		} else {
			result.Spec = obj.Spec
			if _, err = executorsClient.Update(result); err != nil {
				return images, err
			}
		}
	}

	return images, nil
}

// GetPodErrorMessage return pod error message
func GetPodErrorMessage(pod *corev1.Pod) string {
	if pod.Status.Message != "" {
		return pod.Status.Message
	}

	for _, initContainerStatus := range pod.Status.InitContainerStatuses {
		if initContainerStatus.State.Terminated != nil && initContainerStatus.State.Terminated.Message != "" {
			return initContainerStatus.State.Terminated.Message
		}
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.Message != "" {
			return containerStatus.State.Terminated.Message
		}
	}

	return ""
}
