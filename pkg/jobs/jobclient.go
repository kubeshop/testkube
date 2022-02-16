package jobs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/secret"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	pollTimeout  = 24 * time.Hour
	pollInterval = 200 * time.Millisecond

	volumeName = "data-volume"
	volumeDir  = "/data"
)

type JobClient struct {
	ClientSet  *kubernetes.Clientset
	Repository result.Repository
	Namespace  string
	Cmd        string
	Log        *zap.SugaredLogger
}

func NewJobClient() (*JobClient, error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, err
	}

	return &JobClient{
		ClientSet: clientSet,
		Namespace: "testkube",
		Log:       log.DefaultLogger,
	}, nil
}

// LaunchK8sJobSync launches new job and run executor of given type
// TODO Consider moving launch of K8s job as always sync
// TODO Consider moving storage calls level up (remove dependency from here)
func (c *JobClient) LaunchK8sJobSync(image string, repo result.Repository, execution testkube.Execution,
	hasSecrets bool) (result testkube.ExecutionResult, err error) {
	result = testkube.NewPendingExecutionResult()

	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)
	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	ctx := context.Background()

	jsn, err := json.Marshal(execution)
	if err != nil {
		return result.Err(err), err
	}

	jobSpec := NewJobSpec(execution.Id, c.Namespace, image, string(jsn), execution.ScriptName, hasSecrets)

	_, err = jobs.Create(ctx, jobSpec, metav1.CreateOptions{})
	if err != nil {
		return result.Err(err), err
	}

	pods, err := c.GetJobPods(podsClient, execution.Id, 1, 10)
	if err != nil {
		return result.Err(err), err
	}

	// get job pod and
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			l := c.Log.With("pod", pod.Name, "namespace", pod.Namespace, "func", "LaunchK8sJobSync")

			// save stop time
			defer func() {
				execution.Stop()
				repo.EndExecution(ctx, execution.Id, execution.EndTime, execution.CalculateDuration())
			}()

			// wait for complete
			l.Debug("poll immediate waiting for pod to succeed")
			if err := wait.PollImmediate(pollInterval, pollTimeout, IsPodReady(c.ClientSet, pod.Name, c.Namespace)); err != nil {
				// continue on poll err and try to get logs later
				l.Errorw("waiting for pod complete error", "error", err)
			}
			l.Debug("poll immediate end")

			var logs []byte
			logs, err = c.GetPodLogs(pod.Name)
			if err != nil {
				l.Errorw("get pod logs error", "error", err)
				repo.UpdateResult(ctx, execution.Id, result.Err(err))
				return result, err
			}

			// parse job ouput log (JSON stream)
			result, _, err := output.ParseRunnerOutput(logs)
			if err != nil {
				l.Errorw("parse ouput error", "error", err)
				repo.UpdateResult(ctx, execution.Id, result.Err(err))
				return result, err
			}

			l.Infow("execution completed saving result", "executionId", execution.Id, "status", result.Status)
			repo.UpdateResult(ctx, execution.Id, result)
			return result, nil
		}
	}

	return
}

// LaunchK8sJob launches new job and run executor of given type
// TODO consider moving storage based operation up in hierarchy
// TODO Consider moving launch of K8s job as always sync
func (c *JobClient) LaunchK8sJob(image string, repo result.Repository, execution testkube.Execution, hasSecrets bool) (
	result testkube.ExecutionResult, err error) {

	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)
	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	ctx := context.Background()

	// init result
	result = testkube.NewPendingExecutionResult()

	jsn, err := json.Marshal(execution)
	if err != nil {
		return result.Err(err), err
	}

	jobSpec := NewJobSpec(execution.Id, c.Namespace, image, string(jsn), execution.ScriptName, hasSecrets)

	_, err = jobs.Create(ctx, jobSpec, metav1.CreateOptions{})
	if err != nil {
		return result.Err(err), err
	}

	pods, err := c.GetJobPods(podsClient, execution.Id, 1, 10)
	if err != nil {
		return result.Err(err), err
	}

	// get job pod and
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Labels["job-name"] == execution.Id {
			// async wait for complete status or error
			go func() {
				l := c.Log.With("executionID", execution.Id, "func", "LaunchK8sJob")
				// save stop time
				defer func() {
					l.Debug("stopping execution")
					execution.Stop()
					repo.EndExecution(ctx, execution.Id, execution.EndTime, execution.CalculateDuration())
				}()

				// wait for complete
				l.Debug("poll immediate waiting for pod to succeed")
				if err := wait.PollImmediate(pollInterval, pollTimeout, IsPodReady(c.ClientSet, pod.Name, c.Namespace)); err != nil {
					// continue on poll err and try to get logs later
					l.Errorw("poll immediate error", "error", err)
				}
				l.Debug("poll immediate end")

				var logs []byte
				logs, err = c.GetPodLogs(pod.Name)
				if err != nil {
					l.Errorw("get pod logs error", "error", err)
					repo.UpdateResult(ctx, execution.Id, result.Err(err))
					return
				}

				// parse job ouput log (JSON stream)
				result, _, err := output.ParseRunnerOutput(logs)
				if err != nil {
					l.Errorw("parse ouput error", "error", err)
					repo.UpdateResult(ctx, execution.Id, result.Err(err))
					return
				}

				l.Infow("execution completed saving result", "status", result.Status)
				repo.UpdateResult(ctx, execution.Id, result)
			}()
		}
	}

	return testkube.NewPendingExecutionResult(), nil
}

func (c *JobClient) GetJobPods(podsClient tcorev1.PodInterface, jobName string, retryNr, retryCount int) (*corev1.PodList, error) {
	pods, err := podsClient.List(context.TODO(), metav1.ListOptions{LabelSelector: "job-name=" + jobName})
	if err != nil {
		return nil, err
	}
	if retryNr == retryCount {
		return nil, fmt.Errorf("retry count exceeeded, there are no active pods with given id=%s", jobName)
	}
	if len(pods.Items) == 0 {
		time.Sleep(time.Duration(retryNr * 500 * int(time.Millisecond))) // increase backoff timeout
		return c.GetJobPods(podsClient, jobName, retryNr+1, retryCount)
	}
	return pods, nil
}

// TailJobLogs - locates logs for job pod(s)
func (c *JobClient) TailJobLogs(id string, logs chan []byte) (err error) {

	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	ctx := context.Background()

	pods, err := c.GetJobPods(podsClient, id, 1, 10)
	if err != nil {
		close(logs)
		return err
	}

	for _, pod := range pods.Items {
		if pod.Labels["job-name"] == id {

			l := c.Log.With("podNamespace", pod.Namespace, "podName", pod.Name, "podStatus", pod.Status)

			switch pod.Status.Phase {

			case corev1.PodRunning:
				l.Debug("tailing pod logs: immediately")
				return c.TailPodLogs(ctx, pod.Name, logs)

			case corev1.PodFailed:
				err := fmt.Errorf("can't get pod logs, pod failed: %s/%s", pod.Namespace, pod.Name)
				l.Errorw(err.Error())
				return c.GetLastLogLineError(ctx, pod.Namespace, pod.Name)

			default:
				l.Debugw("tailing job logs: waiting for pod to be ready")
				if err = wait.PollImmediate(pollInterval, pollTimeout, IsPodReady(c.ClientSet, pod.Name, c.Namespace)); err != nil {
					l.Errorw("poll immediate error when tailing logs", "error", err)
					return c.GetLastLogLineError(ctx, pod.Namespace, pod.Name)
				}

				l.Debug("tailing pod logs")
				return c.TailPodLogs(ctx, pod.Name, logs)
			}
		}
	}

	return
}

func (c *JobClient) GetLastLogLineError(ctx context.Context, podNamespace, podName string) error {
	l := c.Log.With("pod", podName, "namespace", podNamespace)
	log, err := c.GetPodLogError(ctx, podName)
	if err != nil {
		return fmt.Errorf("getPodLogs error: %w", err)
	}

	l.Debugw("log", "got last log bytes", string(log)) // in case distorted log bytes
	entry, err := output.GetLogEntry(log)
	if err != nil {
		return fmt.Errorf("GetLogEntry error: %w", err)
	}

	c.Log.Errorw("got last log entry", "log", entry.String())
	return fmt.Errorf("error from last log entry: %s", entry.String())
}

func (c *JobClient) GetPodLogs(podName string, logLinesCount ...int64) (logs []byte, err error) {
	count := int64(100)
	if len(logLinesCount) > 0 {
		count = logLinesCount[0]
	}

	podLogOptions := corev1.PodLogOptions{
		Follow:    false,
		TailLines: &count,
	}

	podLogRequest := c.ClientSet.CoreV1().
		Pods(c.Namespace).
		GetLogs(podName, &podLogOptions)

	stream, err := podLogRequest.Stream(context.TODO())
	if err != nil {
		return logs, err
	}

	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return logs, err
	}

	return buf.Bytes(), nil
}

func (c *JobClient) GetPodLogError(ctx context.Context, podName string) (logsBytes []byte, err error) {
	// error line should be last one
	return c.GetPodLogs(podName, 1)
}

func (c *JobClient) TailPodLogs(ctx context.Context, podName string, logs chan []byte) (err error) {
	count := int64(1)

	podLogOptions := corev1.PodLogOptions{
		Follow:    true,
		TailLines: &count,
	}

	podLogRequest := c.ClientSet.CoreV1().
		Pods(c.Namespace).
		GetLogs(podName, &podLogOptions)

	stream, err := podLogRequest.Stream(ctx)
	if err != nil {
		return err
	}

	go func() {
		defer close(logs)

		scanner := bufio.NewScanner(stream)

		// set default bufio scanner buffer (to limit bufio.Scanner: token too long errors on very long lines)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			c.Log.Debug("TailPodLogs stream scan", "out", scanner.Text(), "pod", podName)
			logs <- scanner.Bytes()
		}

		if scanner.Err() != nil {
			c.Log.Errorw("scanner error", "error", scanner.Err())
		}
	}()
	return
}

func (c *JobClient) AbortK8sJob(jobName string) *testkube.ExecutionResult {
	var zero int64 = 0
	bg := metav1.DeletePropagationBackground
	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)
	err := jobs.Delete(context.TODO(), jobName, metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
		PropagationPolicy:  &bg,
	})
	if err != nil {
		return &testkube.ExecutionResult{
			Status: testkube.ExecutionStatusError,
			Output: err.Error(),
		}
	}
	return &testkube.ExecutionResult{
		Status: testkube.ExecutionStatusSuccess,
	}
}

func (c *JobClient) CreatePersistentVolume(name string) error {
	quantity, err := resource.ParseQuantity("10Gi")
	if err != nil {
		return err
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"type": "local"},
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity:    corev1.ResourceList{"storage": quantity},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: fmt.Sprintf("/mnt/data/%s", name),
				},
			},
			StorageClassName: "manual",
		},
	}

	if _, err = c.ClientSet.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *JobClient) CreatePersistentVolumeClaim(name string) error {
	storageClassName := "manual"
	quantity, err := resource.ParseQuantity("10Gi")
	if err != nil {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"storage": quantity},
			},
		},
	}

	if _, err := c.ClientSet.CoreV1().PersistentVolumeClaims(c.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

// NewJobSpec is a method to create new job spec
func NewJobSpec(id, namespace, image, jsn, scriptName string, hasSecrets bool) *batchv1.Job {
	var TTLSecondsAfterFinished int32 = 180
	// TODO backOff need to be handled correctly by Logs and by Running job spec - currently we can get unexpected results
	var backOffLimit int32 = 0

	var secretEnvVars []corev1.EnvVar
	if hasSecrets {
		secretEnvVars = []corev1.EnvVar{
			{
				Name: GitUsernameEnvVarName,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secret.GetMetadataName(scriptName),
						},
						Key: GitUsernameSecretName,
					},
				},
			},
			{
				Name: GitTokenEnvVarName,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secret.GetMetadataName(scriptName),
						},
						Key: GitTokenSecretName,
					},
				},
			},
		}
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &TTLSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:            id + "-init",
							Image:           image,
							Command:         []string{"/bin/runner", jsn},
							ImagePullPolicy: corev1.PullAlways,
							Env:             append(envVars, secretEnvVars...),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: volumeDir,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            id,
							Image:           image,
							Command:         []string{"/bin/runner", jsn},
							ImagePullPolicy: corev1.PullAlways,
							Env:             append(envVars, secretEnvVars...),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: volumeDir,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}
}

var envVars = []corev1.EnvVar{
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
		Value: volumeDir,
	},
}

// IsPodReady defines if pod is ready or failed for logs scrapping
func IsPodReady(c *kubernetes.Clientset, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			return true, nil
		case corev1.PodFailed:
			return true, fmt.Errorf("pod %s/%s failed", pod.Namespace, pod.Name)
		}
		return false, nil
	}
}
