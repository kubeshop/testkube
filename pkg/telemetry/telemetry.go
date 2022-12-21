package telemetry

import (
	"net/http"
	"sync"

	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	httpclient "github.com/kubeshop/testkube/pkg/http"
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
	payload := NewAPIPayload(clusterId, "testkube_api_start", version, "localhost")
	return sendData(senders, payload)
}

// SendCmdEvent will send CLI event to GA
func SendCmdEvent(cmd *cobra.Command, version string) (string, error) {
	// get all sub-commands passed to cli
	command := strings.TrimPrefix(cmd.CommandPath(), "kubectl-testkube ")
	if command == "" {
		command = "root"
	}

	payload := NewCLIPayload(getCurrentContext(), GetMachineID(), command, version, "cli_command_execution")
	return sendData(senders, payload)
}

// SendCmdInitEvent will send CLI event to GA
func SendCmdInitEvent(cmd *cobra.Command, version string) (string, error) {
	payload := NewCLIPayload(getCurrentContext(), GetMachineID(), "init", version, "cli_command_execution")
	return sendData(senders, payload)
}

// SendHeartbeatEvent will send CLI event to GA
func SendHeartbeatEvent(host, version, clusterId string) (string, error) {
	payload := NewAPIPayload(clusterId, "testkube_api_heartbeat", version, host)
	return sendData(senders, payload)
}

// SendCreateEvent will send API create event for Test or Test suite to GA
func SendCreateEvent(event string, params CreateParams) (string, error) {
	payload := NewCreatePayload(event, params)
	return sendData(senders, payload)
}

// SendCreateEvent will send API run event for Test or Test suite to GA
func SendRunEvent(event string, params RunParams) (string, error) {
	payload := NewRunPayload(event, params)
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
		OrganizationId: data.CloudContext.Organization,
		EnvironmentId:  data.CloudContext.Environment,
	}
}
