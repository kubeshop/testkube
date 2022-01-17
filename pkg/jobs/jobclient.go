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
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/runner/output"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	pods "k8s.io/client-go/kubernetes/typed/core/v1"
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

// LaunchK8sJob launches new job and run executor of given type
// TODO Consider moving launch of K8s job as always sync
// TODO Consider moving storage calls level up (remove dependency from here)
func (c *JobClient) LaunchK8sJobSync(image string, repo result.Repository, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	result = testkube.NewPendingExecutionResult()

	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)
	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	ctx := context.Background()

	jsn, err := json.Marshal(execution)
	if err != nil {
		return result.Err(err), err
	}

	jobSpec := NewJobSpec(execution.Id, c.Namespace, image, string(jsn))

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
		if pod.Status.Phase != v1.PodRunning && pod.Labels["job-name"] == execution.Id {

			// save stop time
			defer func() {
				execution.Stop()
				repo.EndExecution(ctx, execution.Id, execution.EndTime, execution.CalculateDuration())
			}()

			// wait for complete
			if err := wait.PollImmediate(time.Second, time.Duration(0)*time.Second, k8sclient.HasPodSucceeded(c.ClientSet, pod.Name, c.Namespace)); err != nil {
				c.Log.Errorw("poll immediate error", "error", err)
				repo.UpdateResult(ctx, execution.Id, result.Err(err))
				return result, err
			}

			var logs []byte
			logs, err = c.GetPodLogs(pod.Name)
			if err != nil {
				c.Log.Errorw("get pod logs error", "error", err)
				repo.UpdateResult(ctx, execution.Id, result.Err(err))
				return
			}

			// parse job ouput log (JSON stream)
			result, _, err := output.ParseRunnerOutput(logs)
			if err != nil {
				c.Log.Errorw("parse ouput error", "error", err)
				repo.UpdateResult(ctx, execution.Id, result.Err(err))
				return result, err
			}

			c.Log.Infow("execution completed saving result", "executionId", execution.Id, "status", result.Status)
			repo.UpdateResult(ctx, execution.Id, result)
			return result, nil
		}
	}

	return
}

// LaunchK8sJob launches new job and run executor of given type
// TODO consider moving storage based operation up in hierarchy
// TODO Consider moving launch of K8s job as always sync
func (c *JobClient) LaunchK8sJob(image string, repo result.Repository, execution testkube.Execution) (result testkube.ExecutionResult, err error) {

	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)
	podsClient := c.ClientSet.CoreV1().Pods(c.Namespace)
	ctx := context.Background()

	jsn, err := json.Marshal(execution)
	if err != nil {
		return result.Err(err), err
	}

	jobSpec := NewJobSpec(execution.Id, c.Namespace, image, string(jsn))

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
		if pod.Status.Phase != v1.PodRunning && pod.Labels["job-name"] == execution.Id {
			// async wait for complete status or error
			go func() {
				// save stop time
				defer func() {
					execution.Stop()
					repo.EndExecution(ctx, execution.Id, execution.EndTime, execution.CalculateDuration())
				}()
				// wait for complete
				if err := wait.PollImmediate(time.Second, time.Duration(0)*time.Second, k8sclient.HasPodSucceeded(c.ClientSet, pod.Name, c.Namespace)); err != nil {
					c.Log.Errorw("poll immediate error", "error", err)
					repo.UpdateResult(ctx, execution.Id, result.Err(err))
					return
				}

				var logs []byte
				logs, err = c.GetPodLogs(pod.Name)
				if err != nil {
					c.Log.Errorw("get pod logs error", "error", err)
					repo.UpdateResult(ctx, execution.Id, result.Err(err))
					return
				}

				// parse job ouput log (JSON stream)
				result, _, err := output.ParseRunnerOutput(logs)
				if err != nil {
					c.Log.Errorw("parse ouput error", "error", err)
					repo.UpdateResult(ctx, execution.Id, result.Err(err))
					return
				}

				c.Log.Infow("execution completed saving result", "executionId", execution.Id, "status", result.Status)
				repo.UpdateResult(ctx, execution.Id, result)
			}()
		}
	}

	return testkube.NewPendingExecutionResult(), nil
}

func (c *JobClient) GetJobPods(podsClient pods.PodInterface, jobName string, retryNr, retryCount int) (*v1.PodList, error) {
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
			if pod.Status.Phase != v1.PodRunning {
				c.Log.Debugw("Waiting for pod to be ready", "pod", pod.Name)
				if err = wait.PollImmediate(100*time.Millisecond, time.Duration(0)*time.Second, k8sclient.IsPodReady(c.ClientSet, pod.Name, c.Namespace)); err != nil {
					c.Log.Errorw("poll immediate error when tailing logs", "error", err)
					close(logs)
					return err
				}
				c.Log.Debug("Tailing pod logs")
				return c.TailPodLogs(ctx, pod.Name, logs)
			} else if pod.Status.Phase == v1.PodRunning {
				return c.TailPodLogs(ctx, pod.Name, logs)
			}
		}
	}

	return
}

func (c *JobClient) GetPodLogs(podName string) (logs []byte, err error) {
	count := int64(100)

	podLogOptions := v1.PodLogOptions{
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

func (c *JobClient) TailPodLogs(ctx context.Context, podName string, logs chan []byte) (err error) {
	count := int64(1)

	podLogOptions := v1.PodLogOptions{
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
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"type": "local"},
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity:    v1.ResourceList{"storage": quantity},
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				HostPath: &v1.HostPathVolumeSource{
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

	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{"storage": quantity},
			},
		},
	}

	if _, err := c.ClientSet.CoreV1().PersistentVolumeClaims(c.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func NewJobSpec(id, namespace, image, jsn string) *batchv1.Job {
	var TTLSecondsAfterFinished int32 = 180
	var backOffLimit int32 = 2

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &TTLSecondsAfterFinished,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            id,
							Image:           image,
							Command:         []string{"/bin/runner", jsn},
							ImagePullPolicy: v1.PullAlways,
							Env:             envVars,
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}

}

var envVars = []v1.EnvVar{
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
}
