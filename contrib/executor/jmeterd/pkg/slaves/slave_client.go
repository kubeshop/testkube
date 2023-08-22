package slaves

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/contrib/executor/jmeterd/pkg/jmeter_env"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	serverPort = 1099
	localport  = 60001
)

type Interface interface {
	CreateSlaves(replicaCount int) error
	DeleteSlaves(podName string) error
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

// creating slaves as per count provided in the SLAVES_CLOUNT env variable.
// Default SLAVES_COUNT would be 1 if not provided in the env variables
func (client *Client) CreateSlaves() (map[string]string, error) {
	slavesCount, err := getSlavesCount(client.envVariables[jmeter_env.SLAVES_COUNT])
	if err != nil {
		return nil, errors.Errorf("Getting error while fetching slaves count from env variable SLAVES_COUNT : %v", err)
	}

	output.PrintLog(fmt.Sprintf("Creating Slaves %v Pods", slavesCount))

	podIPAddressChan := make(chan map[string]string, slavesCount)
	errorChan := make(chan error, slavesCount)
	podIPAddresses := make(map[string]string)

	for i := 1; i <= slavesCount; i++ {
		go client.createSlavePod(i, podIPAddressChan, errorChan)
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


//created slaves pod and send its ipaddress on the podIPAddressChan channel when pod is in the ready state
func (client *Client) createSlavePod(currentSlavesCount int, podIPAddressChan chan<- map[string]string, errorChan chan<- error) {
	runnerExecution, err := json.Marshal(client.execution)
	if err != nil {
		errorChan <- err
		return
	}

	slavePod := getSlavePodConfiguration(client.execution.Name, string(runnerExecution), client.envVariables, client.envParams)
	slavePod.Name = fmt.Sprintf("%s-%v-%v", slavePod.Name, currentSlavesCount, client.execution.Id)

	p, err := client.clientSet.CoreV1().Pods(client.namespace).Create(context.Background(), slavePod, metav1.CreateOptions{})
	if err != nil {
		errorChan <- err
		return
	}

	// Wait for the pod to become ready
	conditionFunc := isPodReady(context.Background(), client.clientSet, p.Name, client.namespace)
	timeout := 5 * time.Minute

	err = wait.PollImmediate(time.Second, timeout, conditionFunc)
	if err != nil {
		errorChan <- err
		return
	}

	p, err = client.clientSet.CoreV1().Pods(client.namespace).Get(context.Background(), p.Name, metav1.GetOptions{})
	if err != nil {
		errorChan <- err
		return
	}
	podNameIpMap := map[string]string{
		p.Name: p.Status.PodIP,
	}
	podIPAddressChan <- podNameIpMap
}

func (client *Client) DeleteSlaves(slaveNameIpMap map[string]string) {
	for slaveName := range slaveNameIpMap {
		output.PrintLog(fmt.Sprintf("Deleting slave %v", slaveName))
		client.clientSet.CoreV1().Pods(client.namespace).Delete(context.Background(), slaveName, metav1.DeleteOptions{})
	}
}
