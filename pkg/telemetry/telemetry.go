package telemetry

import (
	"context"
	"net/http"
	"sync"

	"strings"

	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	httpclient "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
)

var (
	client  = httpclient.NewClient()
	senders = map[string]Sender{
		"google":    GoogleAnalyticsSender,
		"testkube":  TestkubeAnalyticsSender,
		"segmentio": SegmentioSender,
	}
)

type Sender func(client *http.Client, payload Payload) (out string, err error)

// SendServerStartEvent will send event to GA
func SendServerStartEvent(clusterId, version string) (string, error) {
	payload := NewAPIPayload(clusterId, "testkube_api_start", version, "localhost", GetClusterType())
	return sendData(senders, payload)
}

// SendCmdEvent will send CLI event to GA
func SendCmdEvent(cmd *cobra.Command, version string) (string, error) {
	// get all sub-commands passed to cli
	command := strings.TrimPrefix(cmd.CommandPath(), "kubectl-testkube ")
	if command == "" {
		command = "root"
	}

	payload := NewCLIPayload(getCurrentContext(), GetMachineID(), command, version, "cli_command_execution", GetClusterType())
	return sendData(senders, payload)
}

// SendCmdInitEvent will send CLI event to GA
func SendCmdInitEvent(cmd *cobra.Command, version string) (string, error) {
	payload := NewCLIPayload(getCurrentContext(), GetMachineID(), "init", version, "cli_command_execution", GetClusterType())
	return sendData(senders, payload)
}

// SendHeartbeatEvent will send CLI event to GA
func SendHeartbeatEvent(host, version, clusterId string) (string, error) {
	payload := NewAPIPayload(clusterId, "testkube_api_heartbeat", version, host, GetClusterType())
	return sendData(senders, payload)
}

// SendCreateEvent will send API create event for Test or Test suite to GA
func SendCreateEvent(event string, params CreateParams) (string, error) {
	payload := NewCreatePayload(event, GetClusterType(), params)
	return sendData(senders, payload)
}

// SendCreateEvent will send API run event for Test or Test suite to GA
func SendRunEvent(event string, params RunParams) (string, error) {
	payload := NewRunPayload(event, GetClusterType(), params)
	return sendData(senders, payload)
}

// sendData sends data to all telemetry storages  in parallel and syncs sending
func sendData(senders map[string]Sender, payload Payload) (out string, err error) {
	var wg sync.WaitGroup
	wg.Add(len(senders))
	for name, sender := range senders {
		go func(sender Sender, name string) {
			defer wg.Done()
			o, err := sender(client, payload)
			if err != nil {
				log.DefaultLogger.Debugw("sending telemetry data error", "payload", payload, "error", err.Error())
				return
			}
			log.DefaultLogger.Debugw("sending telemetry data", "payload", payload, "output", o, "sender", name)
		}(sender, name)
	}

	wg.Wait()

	return out, nil
}

func getCurrentContext() RunContext {
	data, err := config.Load()
	if err != nil {
		return RunContext{
			Type: "invalid-context",
		}
	}
	return RunContext{
		Type:           string(data.ContextType),
		OrganizationId: data.CloudContext.OrganizationId,
		EnvironmentId:  data.CloudContext.EnvironmentId,
	}
}

func GetClusterType() string {

	clientset, err := k8sclient.ConnectToK8s()
	if err != nil {
		log.DefaultLogger.Debugw("Creating k8s clientset", err)
		return "unidentified"
	}

	pods, err := clientset.CoreV1().Pods("kube-system").List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.DefaultLogger.Debugw("Getting pods from kube-system namespace", err)
		return "unidentified"
	}

	// Loop through the pods and check if their name contains the search string.
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "-kind-") || strings.Contains(pod.Name, "kindnet") {
			return "kind"
		}
		if strings.Contains(pod.Name, "-minikube") {
			return "minikube"
		}
		if strings.Contains(pod.Name, "docker-desktop") {
			return "docker-desktop"
		}
		if strings.Contains(pod.Name, "gke-") || strings.Contains(pod.Name, "-gke-") {
			return "gke"
		}
		if strings.Contains(pod.Name, "aws-") || strings.Contains(pod.Name, "-aws-") {
			return "eks"
		}
		if strings.Contains(pod.Name, "azure-") || strings.Contains(pod.Name, "-azuredisk-") || strings.Contains(pod.Name, "-azurefile-") {
			return "aks"
		}
		if strings.Contains(pod.Name, "openshift") || strings.Contains(pod.Name, "oc-") {
			return "openshift"
		}
		if strings.Contains(pod.Name, "k3d-") {
			return "k3d"
		}
		if strings.Contains(pod.Name, "k3s-") {
			return "k3s"
		}
		if strings.Contains(pod.Name, "microk8s-") {
			return "microk8s"
		}
	}

	return "others"
}
