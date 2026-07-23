package telemetry

import (
	"net/http"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/cliruntime"
	httpclient "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

var (
	client  = httpclient.NewClient()
	senders = map[string]Sender{
		"google":    GoogleAnalyticsSender,
		"segmentio": SegmentioSender,
	}
)

type Sender func(client *http.Client, payload Payload) (out string, err error)

// SendServerStartEvent will send event to GA
func SendServerStartEvent(clusterId, version string, capabilities []string) (string, error) {
	payload := NewAPIPayload(clusterId, "testkube_api_start", version, "localhost", GetClusterType(), capabilities)
	return sendData(senders, payload)
}

// SendCmdEvent will send CLI event to GA
func SendCmdEvent(cmd *cobra.Command, version, userID string) (string, error) {
	// get all sub-commands passed to cli
	command := strings.TrimPrefix(cmd.CommandPath(), "kubectl-testkube ")
	if command == "" {
		command = "root"
	}

	payload := NewCLIPayload(getCurrentContext(), userID, command, version, "cli_command_execution", GetClusterType())
	return sendData(senders, payload)
}

func SendCmdErrorEvent(cmd *cobra.Command, version, errType string, errorStackTrace string) (string, error) {
	return SendCmdErrorEventWithLicense(cmd, version, errType, errorStackTrace, "", "", "")
}

// SendCmdErrorEventWithLicense will send CLI error event with license
func SendCmdErrorEventWithLicense(cmd *cobra.Command, version, errType, errorStackTrace, license, step, errCode string) (string, error) {

	// get all sub-commands passed to cli
	command := strings.TrimPrefix(cmd.CommandPath(), "kubectl-testkube ")
	if command == "" {
		command = "root"
	}

	command += "_error"
	machineID := GetMachineID()
	payload := Payload{
		ClientID: machineID,
		UserID:   machineID,
		Events: []Event{
			{
				Name: text.GAEventName(command),
				Params: Params{
					EventCount:      1,
					EventCategory:   "cli_command_execution",
					AppVersion:      version,
					AppName:         "kubectl-testkube",
					MachineID:       machineID,
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					Context:         getCurrentContext(),
					ClusterType:     GetClusterType(),
					ErrorCode:       errCode,
					ErrorType:       errType,
					ErrorStackTrace: errorStackTrace,
					License:         license,
					Step:            step,
					Email:           GetEmail(license),
				},
			}},
	}

	return sendData(senders, payload)
}

func SendCmdAttemptEvent(cmd *cobra.Command, version, userID string) (string, error) {
	// TODO pass error
	payload := NewCLIPayload(getCurrentContext(), userID, getCommand(cmd), version, "cli_command_execution", GetClusterType())
	return sendData(senders, payload)
}

// SendCmdWithLicenseEvent will send CLI command attempt event with license
func SendCmdWithLicenseEvent(cmd *cobra.Command, version, userID, license, step string) (string, error) {
	payload := NewCLIWithLicensePayload(getCurrentContext(), userID, getCommandWithoutAttempt(cmd), version, "cli_command_execution", GetClusterType(), license, step)
	return sendData(senders, payload)
}

// SendCmdInitEvent will send CLI event to GA
func SendCmdInitEvent(cmd *cobra.Command, version, userID string) (string, error) {
	payload := NewCLIPayload(getCurrentContext(), userID, "init", version, "cli_command_execution", GetClusterType())
	return sendData(senders, payload)
}

// SendTelemetryOptOutEvent records that the user disabled telemetry. It must be
// called while telemetry is still enabled (before the opt-out is persisted), as
// no further events may be sent once the user has opted out.
func SendTelemetryOptOutEvent(cmd *cobra.Command, version, userID string) (string, error) {
	payload := NewCLIPayload(getCurrentContext(), userID, "telemetry_opt_out", version, "cli_command_execution", GetClusterType())
	return sendData(senders, payload)
}

// SendPreviewEvent sends a preview-specific telemetry event with execution context
func SendPreviewEvent(cmd *cobra.Command, version, executionID string, artifactCount int32, skipArtifacts bool, previewErr string) (string, error) {
	machineID := GetMachineID()
	eventName := "preview_execution"
	if previewErr != "" {
		eventName = "preview_execution_error"
	}

	payload := Payload{
		ClientID: machineID,
		UserID:   machineID,
		Events: []Event{
			{
				Name: text.GAEventName(eventName),
				Params: Params{
					EventCount:           1,
					EventCategory:        "cli_command_execution",
					AppVersion:           version,
					AppName:              "kubectl-testkube",
					MachineID:            machineID,
					OperatingSystem:      runtime.GOOS,
					Architecture:         runtime.GOARCH,
					Context:              getCurrentContext(),
					ClusterType:          GetClusterType(),
					CliContext:           GetCliRunContext(),
					PreviewExecutionID:   executionID,
					PreviewArtifacts:     artifactCount,
					PreviewSkipArtifacts: skipArtifacts,
					PreviewError:         previewErr,
				},
			},
		},
	}
	return sendData(senders, payload)
}

// SendHeartbeatEvent will send event to GA
func SendHeartbeatEvent(host, version, clusterId string, capabilities []string) (string, error) {
	payload := NewAPIPayload(clusterId, "testkube_api_heartbeat", version, host, GetClusterType(), capabilities)
	return sendData(senders, payload)
}

// SendCreateEvent will send API create event for Test or Test suite to GA
func SendCreateEvent(event string, params CreateParams) (string, error) {
	payload := NewCreatePayload(event, GetClusterType(), params)
	return sendData(senders, payload)
}

// SendRunEvent will send API run event for Test, or Test suite to GA
func SendRunEvent(event string, params RunParams) (string, error) {
	payload := NewRunPayload(event, GetClusterType(), params)
	return sendData(senders, payload)
}

// SendCreateWorkflowEvent will send API create event for Test workflows to GA
func SendCreateWorkflowEvent(event string, params CreateWorkflowParams) (string, error) {
	payload := NewCreateWorkflowPayload(event, GetClusterType(), params)
	return sendData(senders, payload)
}

// SendRunWorkflowEvent will send API run event for Test workflows to GA
func SendRunWorkflowEvent(event string, params RunWorkflowParams) (string, error) {
	payload := NewRunWorkflowPayload(event, GetClusterType(), params)
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

func getCommand(cmd *cobra.Command) string {
	return getCommandWithoutAttempt(cmd) + "_attempt"
}

func getCommandWithoutAttempt(cmd *cobra.Command) string {
	command := strings.TrimPrefix(cmd.CommandPath(), "kubectl-testkube ")
	if command == "" {
		command = "root"
	}
	return command
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

// GetCurrentContext returns the current run context, making it accessible from other packages
func GetCurrentContext() RunContext {
	return getCurrentContext()
}

// IsRunningInDocker detects if the CLI is running inside a Docker container.
//
// Kept as a thin wrapper around cliruntime.IsRunningInDocker so external
// callers (this package's tests, existing telemetry consumers) keep working.
func IsRunningInDocker() bool {
	return cliruntime.IsRunningInDocker()
}

// GetDockerContext returns a descriptor of the container environment when
// running inside Docker/Kubernetes, or "" otherwise.
func GetDockerContext() string {
	return cliruntime.DockerContext()
}

// GetCliRunContext returns the CLI runtime context identifier (CI system,
// container runtime, or the local sentinel).
func GetCliRunContext() string {
	return cliruntime.CliRunContext()
}
