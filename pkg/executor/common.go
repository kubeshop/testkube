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

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	executorsmapper "github.com/kubeshop/testkube/pkg/mapper/executors"
	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/utils"
)

var ErrPodInitializing = errors.New("PodInitializing")

const (
	// VolumeDir is volume dir
	VolumeDir            = "/data"
	defaultLogLinesCount = 100
	// SlavesConfigsEnv is slave configs for creating slaves in executor
	SlavesConfigsEnv = "RUNNER_SLAVES_CONFIGS"
)

var RunnerEnvVars = []corev1.EnvVar{
	{
		Name:  "DEBUG",
		Value: getOr("DEBUG", "false"),
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
		Name:  "RUNNER_SKIP_VERIFY",
		Value: getOr("STORAGE_SKIP_VERIFY", "false"),
	},
	{
		Name:  "RUNNER_CERT_FILE",
		Value: os.Getenv("STORAGE_CERT_FILE"),
	},
	{
		Name:  "RUNNER_KEY_FILE",
		Value: os.Getenv("STORAGE_KEY_FILE"),
	},
	{
		Name:  "RUNNER_CA_FILE",
		Value: os.Getenv("STORAGE_CA_FILE"),
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
		Name:  "RUNNER_PRO_MODE",
		Value: getRunnerProMode(),
	},
	{
		Name:  "RUNNER_PRO_API_KEY",
		Value: utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_API_KEY", "TESTKUBE_CLOUD_API_KEY", ""),
	},
	{
		Name:  "RUNNER_PRO_API_TLS_INSECURE",
		Value: utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_TLS_INSECURE", "TESTKUBE_CLOUD_TLS_INSECURE", "false"),
	},
	{
		Name:  "RUNNER_PRO_API_URL",
		Value: utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_URL", "TESTKUBE_CLOUD_URL", ""),
	},
	{
		Name:  "RUNNER_PRO_API_SKIP_VERIFY",
		Value: getOr("TESTKUBE_PRO_SKIP_VERIFY", "false"),
	},
	{
		Name:  "RUNNER_PRO_CONNECTION_TIMEOUT",
		Value: getOr("TESTKUBE_PRO_CONNECTION_TIMEOUT", "10"),
	},
	{
		Name:  "RUNNER_PRO_API_CERT_FILE",
		Value: os.Getenv("TESTKUBE_PRO_CERT_FILE"),
	},
	{
		Name:  "RUNNER_PRO_API_KEY_FILE",
		Value: os.Getenv("TESTKUBE_PRO_KEY_FILE"),
	},
	{
		Name:  "RUNNER_PRO_API_CA_FILE",
		Value: os.Getenv("TESTKUBE_PRO_CA_FILE"),
	},
	{
		Name:  "RUNNER_DASHBOARD_URI",
		Value: os.Getenv("TESTKUBE_DASHBOARD_URI"),
	},
	{
		Name:  "CI",
		Value: "1",
	},
	// DEPRECATED: Use RUNNER_PRO_MODE instead
	{
		Name:  "RUNNER_CLOUD_MODE",
		Value: getRunnerProMode(),
	},
	// DEPRECATED: Use RUNNER_PRO_API_KEY instead
	{
		Name:  "RUNNER_CLOUD_API_KEY",
		Value: utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_API_KEY", "TESTKUBE_CLOUD_API_KEY", ""),
	},
	// DEPRECATED: Use RUNNER_PRO_API_TLS_INSECURE instead
	{
		Name:  "RUNNER_CLOUD_API_TLS_INSECURE",
		Value: utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_TLS_INSECURE", "TESTKUBE_CLOUD_TLS_INSECURE", "false"),
	},
	// DEPRECATED: Use RUNNER_PRO_API_URL instead
	{
		Name:  "RUNNER_CLOUD_API_URL",
		Value: utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_URL", "TESTKUBE_CLOUD_URL", ""),
	},
	// DEPRECATED: Use RUNNER_PRO_API_SKIP_VERIFY instead
	{
		Name:  "RUNNER_CLOUD_API_SKIP_VERIFY",
		Value: getOr("TESTKUBE_PRO_SKIP_VERIFY", "false"),
	},
}

type SlavesConfigs struct {
	Images                SlaveImages             `json:"images"`
	ServiceAccountName    string                  `json:"serviceAccountName"`
	CertificateSecret     string                  `json:"certificateSecret"`
	SlavePodTemplate      string                  `json:"slavePodTemplate"`
	ImagePullSecrets      []string                `json:"imagePullSecrets"`
	EnvConfigMaps         []testkube.EnvReference `json:"envConfigMaps"`
	EnvSecrets            []testkube.EnvReference `json:"envSecrets"`
	ActiveDeadlineSeconds int                     `json:"activeDeadlineSeconds"`
	Features              testkube.Features       `json:"features"`
	NatsUri               string                  `json:"natsUri"`
	LogSidecarImage       string                  `json:"logSidecarImage"`
	RunnerCustomCASecret  string                  `json:"runnerCustomCASecret"`
}

type SlaveImages struct {
	Init     string `json:"init"`
	Slave    string `json:"slave"`
	Registry string `json:"registry"`
}

func GetSlavesConfigs(initImage string,
	slavesMeta executorv1.SlavesMeta,
	registry string,
	serviceAccountName string,
	certificateSecret string,
	slavePodTemplate string,
	imagePullSecrets []string,
	envConfigMaps []testkube.EnvReference,
	envSecrets []testkube.EnvReference,
	activeDeadlineSeconds int,
	features testkube.Features,
	natsUri string,
	logSidecarImage string,
	runnerCustomCASecret string,
) SlavesConfigs {
	return SlavesConfigs{
		Images: SlaveImages{
			Init:     initImage,
			Slave:    slavesMeta.Image,
			Registry: registry,
		},
		ServiceAccountName:    serviceAccountName,
		CertificateSecret:     certificateSecret,
		SlavePodTemplate:      slavePodTemplate,
		ImagePullSecrets:      imagePullSecrets,
		EnvConfigMaps:         envConfigMaps,
		EnvSecrets:            envSecrets,
		ActiveDeadlineSeconds: activeDeadlineSeconds,
		Features:              features,
		NatsUri:               natsUri,
		LogSidecarImage:       logSidecarImage,
		RunnerCustomCASecret:  runnerCustomCASecret,
	}
}

func getOr(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getRunnerProMode() string {
	val := "false"
	if utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_API_KEY", "TESTKUBE_CLOUD_API_KEY", "") != "" {
		val = "true"
	}
	return val
}

// Templates contains templates for executor
type Templates struct {
	Job     string `json:"job"`
	PVC     string `json:"pvc"`
	Scraper string `json:"scraper"`
	Slave   string `json:"slave"`
}

// Images contains images for executor
type Images struct {
	Init       string
	Scraper    string
	LogSidecar string
}

// IsPodReady defines if pod is ready or failed for logs scrapping
func IsPodReady(c kubernetes.Interface, podName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
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
func IsPodLoggable(c kubernetes.Interface, podName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
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

// GetPodByName returns job pod by name
func GetPodByName(ctx context.Context, podsClient tcorev1.PodInterface, podName string, retryNr, retryCount int) (*corev1.Pod, error) {
	pod, err := podsClient.Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if retryNr == retryCount {
		return nil, fmt.Errorf("retry count exceeeded, there is no active pod with given name=%s", podName)
	}
	if pod == nil {
		time.Sleep(time.Duration(retryNr * 500 * int(time.Millisecond))) // increase backoff timeout
		return GetPodByName(ctx, podsClient, podName, retryNr+1, retryCount)
	}
	return pod, nil
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
func GetPodLogs(ctx context.Context, c kubernetes.Interface, namespace string, pod corev1.Pod, id string, whitelistedContainers []string, logLinesCount ...int64) (logs []byte, err error) {
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
		if !IsWhitelistedContainer(container, id, whitelistedContainers) {
			continue
		}
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
) error {
	if len(executors) == 0 {
		return nil
	}

	for _, executor := range executors {
		obj := &executorv1.Executor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      executor.Name,
				Namespace: namespace,
			},
			Spec: executorv1.ExecutorSpec{
				Types:        executor.Executor.Types,
				ExecutorType: executorv1.ExecutorType(executor.Executor.ExecutorType),
				Image:        executor.Executor.Image,
				Slaves:       executorsmapper.MapSlavesConfigsToCRD(executor.Executor.Slaves),
				Command:      executor.Executor.Command,
				Args:         executor.Executor.Args,
				Features:     executorsmapper.MapFeaturesToCRD(executor.Executor.Features),
				ContentTypes: executorsmapper.MapContentTypesToCRD(executor.Executor.ContentTypes),
				Meta:         executorsmapper.MapMetaToCRD(executor.Executor.Meta),
			},
		}

		result, err := executorsClient.Get(executor.Name)
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if err != nil {
			if _, err = executorsClient.Create(obj); err != nil {
				return err
			}
		} else {
			obj.Spec.JobTemplate = result.Spec.JobTemplate
			obj.Spec.JobTemplateReference = result.Spec.JobTemplateReference
			obj.Spec.UseDataDirAsWorkingDir = result.Spec.UseDataDirAsWorkingDir
			result.Spec = obj.Spec
			if _, err = executorsClient.Update(result); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetPodErrorMessage returns pod error message
func GetPodErrorMessage(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod) string {
	message := ""
	if pod.Status.Message != "" || pod.Status.Reason != "" {
		message = fmt.Sprintf("pod message: %s reason: %s", pod.Status.Message, pod.Status.Reason)
	}

	for _, initContainerStatus := range pod.Status.InitContainerStatuses {
		if initContainerStatus.State.Terminated != nil &&
			(initContainerStatus.State.Terminated.ExitCode >= 1 || initContainerStatus.State.Terminated.ExitCode < -1) &&
			(initContainerStatus.State.Terminated.Message != "" || initContainerStatus.State.Terminated.Reason != "") {
			if message != "" {
				message += "\n"
			}

			message += fmt.Sprintf("init container message: %s reason: %s", initContainerStatus.State.Terminated.Message,
				initContainerStatus.State.Terminated.Reason)
			message += fmt.Sprintf("\nexit code: %d", initContainerStatus.State.Terminated.ExitCode)
			return message
		}
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil &&
			(containerStatus.State.Terminated.ExitCode >= 1 || containerStatus.State.Terminated.ExitCode < -1) &&
			(containerStatus.State.Terminated.Message != "" || containerStatus.State.Terminated.Reason != "") {
			if message != "" {
				message += "\n"
			}

			message += fmt.Sprintf("test container message: %s reason: %s", containerStatus.State.Terminated.Message,
				containerStatus.State.Terminated.Reason)
			message += fmt.Sprintf("\nexit code: %d", containerStatus.State.Terminated.ExitCode)
			return message
		}
	}

	if message == "" {
		statusStr := pod.Status.String()
		message = fmt.Sprintf("execution pod %s failed, status: %s", pod.Name, statusStr)
	}

	return message
}

// GetPodExitCode returns pod exit code
func GetPodExitCode(pod *corev1.Pod) int32 {
	for _, initContainerStatus := range pod.Status.InitContainerStatuses {
		if initContainerStatus.State.Terminated != nil && initContainerStatus.State.Terminated.ExitCode != 0 {
			return initContainerStatus.State.Terminated.ExitCode
		}
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
			return containerStatus.State.Terminated.ExitCode
		}
	}

	return 0
}

// GetPodEventsSummary returns pod events summary
func GetPodEventsSummary(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod) (string, error) {
	message := ""
	list, err := client.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, item := range list.Items {
		if item.InvolvedObject.Name != pod.Name {
			continue
		}

		if message != "" {
			message += "\n"
		}

		message += fmt.Sprintf("event type: %s, reason: %s, message: %s",
			item.Type, item.Reason, item.Message)
	}

	return message, nil
}

func IsWhitelistedContainer(containerName string, id string, whitelistedContainers []string) bool {
	if containerName == id {
		return true
	}
	withID := func(name string) string {
		return fmt.Sprintf("%s-%s", id, name)
	}
	for _, whitelistedContainer := range whitelistedContainers {
		if containerName == withID(whitelistedContainer) {
			return true
		}
	}

	return false
}
