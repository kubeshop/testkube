package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type JobClient struct {
	ClientSet *kubernetes.Clientset
	Namespace string
	Cmd       string
}

func NewJobClient() (*JobClient, error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, err
	}

	return &JobClient{
		ClientSet: clientSet,
		Namespace: "default",
	}, nil
}

func (c *JobClient) LaunchK8sJob(jobName string, image string, execution testkube.Execution) (testkube.ExecutionResult, error) {
	jobs := c.ClientSet.BatchV1().Jobs(c.Namespace)
	var result string

	jsn, err := json.Marshal(execution)
	if err != nil {
		return testkube.ExecutionResult{
			Status:       testkube.StatusPtr(testkube.ERROR__ExecutionStatus),
			ErrorMessage: err.Error(),
		}, err
	}

	var TTLSecondsAfterFinished int32 = 1
	var backOffLimit int32 = 2
	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: c.Namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &TTLSecondsAfterFinished,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            jobName,
							Image:           image,
							Command:         []string{"agent", string(jsn)},
							ImagePullPolicy: v1.PullAlways,
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}

	_, err = jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
	if err != nil {
		return testkube.ExecutionResult{
			Status:       testkube.StatusPtr(testkube.ERROR__ExecutionStatus),
			ErrorMessage: err.Error(),
		}, err
	}

	pods, err := c.ClientSet.CoreV1().Pods(c.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "job-name=" + jobName})
	if err != nil {
		return testkube.ExecutionResult{
			Status:       testkube.StatusPtr(testkube.ERROR__ExecutionStatus),
			ErrorMessage: err.Error(),
		}, err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase != v1.PodRunning {
			if pod.Labels["job-name"] == jobName {
				if err := wait.PollImmediate(time.Second, time.Duration(0)*time.Second, k8sclient.IsPodRunning(c.ClientSet, pod.Name, c.Namespace)); err != nil {
					return testkube.ExecutionResult{
						Status:       testkube.StatusPtr(testkube.ERROR__ExecutionStatus),
						ErrorMessage: err.Error(),
					}, err
				}
			}
			result, err = c.GetPodLogs(pod.Name, jobName, jobName)
			if err != nil {
				return testkube.ExecutionResult{
					Status:       testkube.StatusPtr(testkube.ERROR__ExecutionStatus),
					ErrorMessage: err.Error(),
				}, err
			}
		}
	}

	return testkube.ExecutionResult{
		Status: testkube.StatusPtr(testkube.SUCCESS_ExecutionStatus),
		Output: result,
	}, nil
}

func (c *JobClient) GetPodLogs(podName string, containerName string, endMessage string) (string, error) {
	count := int64(100)
	var toReturn string
	var message string
	podLogOptions := v1.PodLogOptions{
		Follow:    true,
		TailLines: &count,
	}

	podLogRequest := c.ClientSet.CoreV1().
		Pods(c.Namespace).
		GetLogs(podName, &podLogOptions)
	stream, err := podLogRequest.Stream(context.TODO())
	if err != nil {
		return "", err
	}

	defer stream.Close()

	for {
		buf := make([]byte, 2000)
		numBytes, err := stream.Read(buf)
		if numBytes == 0 {
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		message = string(buf[:numBytes])
		if strings.Contains(message, fmt.Sprintf("$$$%s$$$", endMessage)) {
			message = ""
			break
		} else {
			toReturn += message
		}
	}
	return toReturn, nil
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
			Status: testkube.StatusPtr(testkube.ERROR__ExecutionStatus),
			Output: err.Error(),
		}
	}
	return &testkube.ExecutionResult{
		Status: testkube.StatusPtr(testkube.SUCCESS_ExecutionStatus),
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
