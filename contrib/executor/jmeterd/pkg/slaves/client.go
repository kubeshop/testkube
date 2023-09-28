package slaves

import (
	"context"
	"encoding/json"
	"time"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/contrib/executor/jmeterd/pkg/jmeterenv"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/k8sclient"
)

const (
	podsTimeout = 5 * time.Minute
	job         = "Job"
	batchV1     = "batch/v1"
)

type Client struct {
	clientSet     *kubernetes.Clientset
	slavesConfigs executor.SlavesConfigs
	namespace     string
	execution     testkube.Execution
	envParams     envs.Params
	envVariables  map[string]testkube.Variable
}

// NewClient is a method to create new slave client
func NewClient(execution testkube.Execution, slavesConfigs executor.SlavesConfigs, envParams envs.Params, slavesEnvVariables map[string]testkube.Variable) (*Client, error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, err
	}

	return &Client{
		clientSet:     clientSet,
		slavesConfigs: slavesConfigs,
		namespace:     execution.TestNamespace,
		execution:     execution,
		envParams:     envParams,
		envVariables:  slavesEnvVariables,
	}, nil
}

// CreateSlaves creates slaves as per count provided in the SLAVES_COUNT env variable.
// Default SLAVES_COUNT would be 1 if not provided in the env variables
func (c *Client) CreateSlaves(ctx context.Context) (SlaveMeta, error) {
	slavesCount, err := getSlavesCount(c.envVariables[jmeterenv.SlavesCount])
	if err != nil {
		return nil, errors.Wrap(err, "error getting slaves count from SLAVES_COUNT environment variable")
	}

	output.PrintLogf("Creating slave pods: %d", slavesCount)
	podIPAddressChan := make(chan map[string]string, slavesCount)
	errorChan := make(chan error, slavesCount)
	podIPAddresses := make(map[string]string)

	for i := 1; i <= slavesCount; i++ {
		go c.createSlavePod(ctx, i, podIPAddressChan, errorChan)
	}

	for i := 0; i < slavesCount; i++ {
		select {
		case ipAddress := <-podIPAddressChan:
			for podName, podIp := range ipAddress {
				podIPAddresses[podName] = podIp
			}
		case err := <-errorChan:
			if err != nil {
				return nil, errors.Wrap(err, "error while creating and resolving slave pod IP addresses")
			}
		}
	}

	output.PrintLog("Successfully resolved slave pods IP addresses")

	slaveMeta := SlaveMeta(podIPAddresses)
	return slaveMeta, nil
}

// createSlavePod creates a slave pod and sends its IP address on the podIPAddressChan
// channel when the pod is in the ready state.
func (c *Client) createSlavePod(ctx context.Context, currentSlavesCount int, podIPAddressChan chan<- map[string]string, errorChan chan<- error) {
	slavePod, err := c.getSlavePodConfiguration(ctx, currentSlavesCount)
	if err != nil {
		errorChan <- err
		return
	}

	p, err := c.clientSet.CoreV1().Pods(c.namespace).Create(ctx, slavePod, metav1.CreateOptions{})
	if err != nil {
		errorChan <- err
		return
	}

	// Wait for the pod to become ready
	conditionFunc := isPodReady(c.clientSet, p.Name, c.namespace)

	if err = wait.PollUntilContextTimeout(ctx, time.Second, podsTimeout, true, conditionFunc); err != nil {
		errorChan <- err
		return
	}

	p, err = c.clientSet.CoreV1().Pods(c.namespace).Get(ctx, p.Name, metav1.GetOptions{})
	if err != nil {
		errorChan <- err
		return
	}
	podNameIPMap := map[string]string{
		p.Name: p.Status.PodIP,
	}
	podIPAddressChan <- podNameIPMap
}

func (c *Client) getSlavePodConfiguration(ctx context.Context, currentSlavesCount int) (*v1.Pod, error) {
	runnerExecutionStr, err := json.Marshal(c.execution)
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling runner execution")
	}

	podName := ValidateAndGetSlavePodName(c.execution.Name, c.execution.Id, currentSlavesCount)
	if err != nil {
		return nil, errors.Wrap(err, "error validating slave pod name")
	}

	executorJob, err := c.clientSet.BatchV1().Jobs(c.namespace).Get(ctx, c.execution.Id, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error getting executor job")
	}

	return c.createSlavePodObject(runnerExecutionStr, podName, executorJob), nil
}

func (c *Client) createSlavePodObject(runnerExecutionStr []byte, podName string, executorJob *batchv1.Job) *v1.Pod {
	labels := map[string]string{
		// Execution ID is the only unique field in case of multiple runs of the same test
		// So this is the only field which can tag the slave pods to actual job of jmeterd executor
		"testkube.io/managed-by": c.execution.Id,
		"testkube.io/test-name":  c.execution.TestName,
	}
	ownerReference := []metav1.OwnerReference{
		{
			Kind:       job,
			APIVersion: batchV1,
			Name:       executorJob.Name,
			UID:        executorJob.UID,
		},
	}
	initContainers := []v1.Container{
		{
			Name:            "init",
			Image:           c.slavesConfigs.Images.Init,
			Command:         []string{"/bin/runner", string(runnerExecutionStr)},
			Env:             getSlaveRunnerEnv(c.envParams, c.execution),
			ImagePullPolicy: v1.PullIfNotPresent,
			VolumeMounts: []v1.VolumeMount{
				{
					MountPath: "/data",
					Name:      "data-volume",
				},
			},
		},
	}
	volumes := []v1.Volume{
		{
			Name:         "data-volume",
			VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
		},
	}
	mainContainerLivenessProbe := &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			TCPSocket: &v1.TCPSocketAction{
				Port: intstr.FromInt32(serverPort),
			},
		},
		FailureThreshold: 3,
		PeriodSeconds:    5,
		SuccessThreshold: 1,
		TimeoutSeconds:   1,
	}
	mainContainerReadinessProbe := &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			TCPSocket: &v1.TCPSocketAction{
				Port: intstr.FromInt32(serverPort),
			},
		},
		FailureThreshold:    3,
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		TimeoutSeconds:      1,
	}
	mainContainerPorts := []v1.ContainerPort{
		{
			ContainerPort: serverPort,
			Name:          "server-port",
		}, {
			ContainerPort: localPort,
			Name:          "local-port",
		},
	}
	mainContainerVolumeMounts := []v1.VolumeMount{
		{
			MountPath: "/data",
			Name:      "data-volume",
		},
	}
	containers := []v1.Container{
		{
			Name:            "main",
			Image:           c.slavesConfigs.Images.Slave,
			Env:             getSlaveConfigurationEnv(c.envVariables),
			ImagePullPolicy: v1.PullIfNotPresent,
			Ports:           mainContainerPorts,
			VolumeMounts:    mainContainerVolumeMounts,
			LivenessProbe:   mainContainerLivenessProbe,
			ReadinessProbe:  mainContainerReadinessProbe,
		},
	}
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            podName,
			Labels:          labels,
			OwnerReferences: ownerReference,
		},
		Spec: v1.PodSpec{
			RestartPolicy:  v1.RestartPolicyAlways,
			InitContainers: initContainers,
			Containers:     containers,
			Volumes:        volumes,
		},
	}
}

func (c *Client) DeleteSlaves(ctx context.Context, meta SlaveMeta) error {
	for _, name := range meta.Names() {
		output.PrintLogf("Deleting slave pod: %v", name)
		err := c.clientSet.CoreV1().Pods(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			output.PrintLogf("Error deleting slave pods: %v", err.Error())
			return err
		}

	}
	return nil
}

var _ Interface = (*Client)(nil)
