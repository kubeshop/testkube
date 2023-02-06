package executor

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	secretenv "github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/log"
	executorsmapper "github.com/kubeshop/testkube/pkg/mapper/executors"
)

var ErrPodInitializing = errors.New("PodInitializing")

const (
	defaultLogLinesCount = 100
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
		Name:  "RUNNER_BUCKET",
		Value: os.Getenv("STORAGE_BUCKET"),
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

// Templates contains templates for executor
type Templates struct {
	Job     string
	PVC     string
	Scraper string
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

		if err = isPodFailed(pod); err != nil {
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
		containerLogs, err := getContainerLogs(ctx, c, &pod, container, namespace, &count)
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

func getContainerLogs(
	ctx context.Context,
	c kubernetes.Interface,
	pod *corev1.Pod,
	container, namespace string,
	tailLines *int64,
) ([]byte, error) {
	podLogOptions := corev1.PodLogOptions{
		Follow:    false,
		TailLines: tailLines,
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

func PrepareEnvs(envs map[string]string) []corev1.EnvVar {
	var env []corev1.EnvVar
	for k, v := range envs {
		env = append(env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	return env
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

// NewTemplatesFromEnv returns base64 encoded templates from nev
func NewTemplatesFromEnv(env string) (t Templates, err error) {
	err = envconfig.Process(env, &t)
	if err != nil {
		return t, err
	}
	templates := []*string{&t.Job, &t.PVC, &t.Scraper}
	for i := range templates {
		if *templates[i] != "" {
			dataDecoded, err := base64.StdEncoding.DecodeString(*templates[i])
			if err != nil {
				return t, err
			}

			*templates[i] = string(dataDecoded)
		}
	}

	return t, nil
}

// SyncDefaultExecutors creates or updates default executors
func SyncDefaultExecutors(executorsClient executorsclientv1.Interface, namespace, data string, readOnlyExecutors bool) (
	images Images, err error) {
	var executors []testkube.ExecutorDetails

	if data == "" {
		return images, nil
	}

	dataDecoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return images, err
	}

	if err := json.Unmarshal(dataDecoded, &executors); err != nil {
		return images, err
	}

	for _, executor := range executors {
		if executor.Executor == nil {
			continue
		}

		if executor.Name == "executor-init" {
			images.Init = executor.Executor.Image
			continue
		}

		if executor.Name == "executor-scraper" {
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
				ExecutorType: executor.Executor.ExecutorType,
				Image:        executor.Executor.Image,
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
