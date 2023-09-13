package slaves

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/contrib/executor/jmeterd/pkg/jmeterenv"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/k8sclient"
)

const (
	podsTimeout = 5 * time.Minute
)

type Interface interface {
	CreateSlaves(ctx context.Context) error
	DeleteSlaves(ctx context.Context, slaveNameIpMap map[string]string) error
}

type Client struct {
	clientSet    *kubernetes.Clientset
	namespace    string
	execution    testkube.Execution
	envParams    envs.Params
	envVariables map[string]testkube.Variable
}

// NewClient is a method to create new slave client
func NewClient(execution testkube.Execution, envParams envs.Params, slavesEnvVariables map[string]testkube.Variable) (*Client, error) {
	clientSet, err := k8sclient.ConnectToK8s()
	if err != nil {
		return nil, err
	}

	return &Client{
		clientSet:    clientSet,
		namespace:    execution.TestNamespace,
		execution:    execution,
		envParams:    envParams,
		envVariables: slavesEnvVariables,
	}, nil
}

// CreateSlaves creates slaves as per count provided in the SLAVES_CLOUNT env variable.
// Default SLAVES_COUNT would be 1 if not provided in the env variables
func (client *Client) CreateSlaves(ctx context.Context) (map[string]string, error) {
	slavesCount, err := getSlavesCount(client.envVariables[jmeterenv.SlavesCount])
	if err != nil {
		return nil, errors.Errorf("Getting error while fetching slaves count from env variable SLAVES_COUNT : %v", err)
	}

	output.PrintLogf("Creating Slaves %v Pods", slavesCount)

	podIPAddressChan := make(chan map[string]string, slavesCount)
	errorChan := make(chan error, slavesCount)
	podIPAddresses := make(map[string]string)

	for i := 1; i <= slavesCount; i++ {
		go client.createSlavePod(ctx, i, podIPAddressChan, errorChan)
	}

	for i := 0; i < slavesCount; i++ {
		select {
		case ipAddress := <-podIPAddressChan:
			for podName, podIp := range ipAddress {
				podIPAddresses[podName] = podIp
			}
		case err := <-errorChan:
			if err != nil {
				return nil, err
			}
		}
	}

	return podIPAddresses, nil
}

// createSlavePod creates a slave pod and sends its IP address on the podIPAddressChan
// channel when the pod is in the ready state.
func (client *Client) createSlavePod(ctx context.Context, currentSlavesCount int, podIPAddressChan chan<- map[string]string, errorChan chan<- error) {
	slavePod, err := client.getSlavePodConfiguration(currentSlavesCount)
	if err != nil {
		errorChan <- err
		return
	}

	p, err := client.clientSet.CoreV1().Pods(client.namespace).Create(ctx, slavePod, metav1.CreateOptions{})
	if err != nil {
		errorChan <- err
		return
	}

	// Wait for the pod to become ready
	conditionFunc := isPodReady(ctx, client.clientSet, p.Name, client.namespace)

	err = wait.PollImmediate(time.Second, podsTimeout, conditionFunc)
	if err != nil {
		errorChan <- err
		return
	}

	p, err = client.clientSet.CoreV1().Pods(client.namespace).Get(ctx, p.Name, metav1.GetOptions{})
	if err != nil {
		errorChan <- err
		return
	}
	podNameIpMap := map[string]string{
		p.Name: p.Status.PodIP,
	}
	podIPAddressChan <- podNameIpMap
}

func (client *Client) getSlavePodConfiguration(currentSlavesCount int) (*v1.Pod, error) {
	runnerExecutionStr, err := json.Marshal(client.execution)
	if err != nil {
		return nil, err
	}

	podName := ValidateAndGetSlavePodName(client.execution.TestName, client.execution.Id, currentSlavesCount)
	if err != nil {
		return nil, err
	}
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyAlways,
			InitContainers: []v1.Container{
				{
					Name:            "init",
					Image:           "kubeshop/testkube-init-executor:1.14.3",
					Command:         []string{"/bin/runner", string(runnerExecutionStr)},
					Env:             getSlaveRunnerEnv(client.envParams, client.execution),
					ImagePullPolicy: v1.PullIfNotPresent,
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: "/data",
							Name:      "data-volume",
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:            "main",
					Image:           "kubeshop/testkube-jmeterd-slaves:999.0.0",
					Env:             getSlaveConfigurationEnv(client.envVariables),
					ImagePullPolicy: v1.PullIfNotPresent,
					Ports: []v1.ContainerPort{
						{
							ContainerPort: serverPort,
							Name:          "server-port",
						}, {
							ContainerPort: localPort,
							Name:          "local-port",
						},
					},
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: "/data",
							Name:      "data-volume",
						},
					},
					LivenessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							TCPSocket: &v1.TCPSocketAction{
								Port: intstr.FromInt(serverPort),
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    5,
						SuccessThreshold: 1,
						TimeoutSeconds:   1,
					},
					ReadinessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							TCPSocket: &v1.TCPSocketAction{
								Port: intstr.FromInt(serverPort),
							},
						},
						FailureThreshold:    3,
						InitialDelaySeconds: 10,
						PeriodSeconds:       5,
						TimeoutSeconds:      1,
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name:         "data-volume",
					VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
				},
			},
		},
	}, nil
}

// DeleteSlaves do the cleanup slaves pods after execution of test
func (client *Client) DeleteSlaves(ctx context.Context, slaveNameIpMap map[string]string) error {
	for slaveName := range slaveNameIpMap {
		output.PrintLog(fmt.Sprintf("Deleting slave %v", slaveName))
		err := client.clientSet.CoreV1().Pods(client.namespace).Delete(ctx, slaveName, metav1.DeleteOptions{})
		if err != nil {
			output.PrintLogf("Error deleting slave pods %v", err.Error())
			return err
		}

	}
	return nil
}
