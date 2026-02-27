package telemetry

import (
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
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
func SendCmdEvent(cmd *cobra.Command, version string) (string, error) {
	// get all sub-commands passed to cli
	command := strings.TrimPrefix(cmd.CommandPath(), "kubectl-testkube ")
	if command == "" {
		command = "root"
	}

	payload := NewCLIPayload(getCurrentContext(), getUserID(cmd), command, version, "cli_command_execution", GetClusterType())
	return sendData(senders, payload)
}

func SendCmdErrorEvent(cmd *cobra.Command, version, errType string, errorStackTrace string) (string, error) {
	return SendCmdErrorEventWithLicense(cmd, version, errType, errorStackTrace, "", "", "")
}

func HandleCLIErrorTelemetry(version string, err *common.CLIError) (string, error) {
	if err.Telemetry != nil {
		return SendCmdErrorEventWithLicense(
			err.Telemetry.Command,
			version,
			err.Telemetry.Type,
			err.StackTrace,
			err.Telemetry.License,
			err.Telemetry.Step,
			string(err.Code),
		)
	}
	return "", nil
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

func SendCmdAttemptEvent(cmd *cobra.Command, version string) (string, error) {
	// TODO pass error
	payload := NewCLIPayload(getCurrentContext(), getUserID(cmd), getCommand(cmd), version, "cli_command_execution", GetClusterType())
	return sendData(senders, payload)
}

// SendCmdWithLicenseEvent will send CLI command attempt event with license
func SendCmdWithLicenseEvent(cmd *cobra.Command, version, license, step string) (string, error) {
	payload := NewCLIWithLicensePayload(getCurrentContext(), getUserID(cmd), getCommandWithoutAttempt(cmd), version, "cli_command_execution", GetClusterType(), license, step)
	return sendData(senders, payload)
}

// SendCmdInitEvent will send CLI event to GA
func SendCmdInitEvent(cmd *cobra.Command, version string) (string, error) {
	payload := NewCLIPayload(getCurrentContext(), getUserID(cmd), "init", version, "cli_command_execution", GetClusterType())
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

func getUserID(cmd *cobra.Command) string {
	id := "command-cli-user"
	client, _, err := common.GetClient(cmd)
	if err == nil && client != nil {
		info, err := client.GetServerInfo()
		if err == nil && info.ClusterId != "" {
			id = info.ClusterId
		}
	}
	data, err := config.Load()
	if err != nil || data.CloudContext.EnvironmentId == "" {
		return id
	}
	return data.CloudContext.EnvironmentId
}

// IsRunningInDocker detects if the CLI is running inside a Docker container
func IsRunningInDocker() bool {
	// Method 1: Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Method 2: Check Docker-specific environment variables
	dockerEnvVars := []string{
		"DOCKER_CONTAINER",
		"DOCKER_BUILDKIT",
		"DOCKER_HOST",
		"DOCKER_TLS_VERIFY",
		"DOCKER_CERT_PATH",
		"DOCKER_MACHINE_NAME",
	}

	for _, envVar := range dockerEnvVars {
		if _, exists := os.LookupEnv(envVar); exists {
			return true
		}
	}

	// Method 3: Check cgroup for Docker/containerd (Linux only)
	if runtime.GOOS == "linux" {
		if cgroupData, err := os.ReadFile("/proc/1/cgroup"); err == nil {
			cgroupContent := string(cgroupData)
			if strings.Contains(cgroupContent, "docker") ||
				strings.Contains(cgroupContent, "containerd") {
				return true
			}
		}
	}

	return false
}

// GetDockerContext detects how the Docker container is being run
func GetDockerContext() string {
	if !IsRunningInDocker() {
		return ""
	}

	// Check for Docker Compose
	if _, ok := os.LookupEnv("COMPOSE_PROJECT_NAME"); ok {
		projectName := os.Getenv("COMPOSE_PROJECT_NAME")
		if projectName != "" {
			return "docker-compose:" + projectName
		}
		return "docker-compose"
	}

	// Check for Docker Swarm
	if _, ok := os.LookupEnv("DOCKER_SWARM"); ok {
		return "docker-swarm"
	}

	// Check for Kubernetes (running in a pod)
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		namespace := os.Getenv("POD_NAMESPACE")
		if namespace != "" {
			return "kubernetes:" + namespace
		}
		return "kubernetes"
	}

	// Check for Docker BuildKit (during build)
	if _, ok := os.LookupEnv("DOCKER_BUILDKIT"); ok {
		return "docker-buildkit"
	}

	// Check for Docker Desktop
	if _, ok := os.LookupEnv("DOCKER_DESKTOP"); ok {
		return "docker-desktop"
	}

	// Check for specific container runtime
	if runtime.GOOS == "linux" {
		if cgroupData, err := os.ReadFile("/proc/1/cgroup"); err == nil {
			cgroupContent := string(cgroupData)
			if strings.Contains(cgroupContent, "containerd") {
				return "containerd"
			}
			if strings.Contains(cgroupContent, "crio") {
				return "cri-o"
			}
		}
	}

	// Check for custom Testkube Docker image version
	if version, ok := os.LookupEnv("TESTKUBE_DOCKER_IMAGE_VERSION"); ok {
		return "docker:testkube:" + version
	}

	// Check for CI/CD environments that might be using Docker
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return "docker:github-actions"
	}
	if _, ok := os.LookupEnv("CIRCLECI"); ok {
		return "docker:circleci"
	}
	if _, ok := os.LookupEnv("GITLAB_CI"); ok {
		return "docker:gitlab-ci"
	}
	if _, ok := os.LookupEnv("BUILDKITE"); ok {
		return "docker:buildkite"
	}

	// Check for container orchestration platforms
	if _, ok := os.LookupEnv("AWS_EXECUTION_ENV"); ok {
		return "docker:aws"
	}
	if _, ok := os.LookupEnv("GOOGLE_CLOUD_PROJECT"); ok {
		return "docker:gcp"
	}
	if _, ok := os.LookupEnv("AZURE_CONTAINER_REGISTRY"); ok {
		return "docker:azure"
	}

	// Default Docker context
	return "docker"
}

func GetCliRunContext() string {
	// Check for Docker first with detailed context
	if dockerContext := GetDockerContext(); dockerContext != "" {
		return dockerContext
	}

	if value, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		if value == "true" {
			return "github-actions"
		}
	}

	if _, ok := os.LookupEnv("TF_BUILD"); ok {
		return "azure-pipelines"
	}

	if _, ok := os.LookupEnv("JENKINS_URL"); ok {
		return "jenkins"
	}

	if _, ok := os.LookupEnv("JENKINS_HOME"); ok {
		return "jenkins"
	}

	if _, ok := os.LookupEnv("CIRCLECI"); ok {
		return "circleci"
	}

	if _, ok := os.LookupEnv("GITLAB_CI"); ok {
		return "gitlab-ci"
	}

	if _, ok := os.LookupEnv("BUILDKITE"); ok {
		return "buildkite"
	}

	if _, ok := os.LookupEnv("TRAVIS"); ok {
		return "travis-ci"
	}

	if _, ok := os.LookupEnv("AIRFLOW_HOME"); ok {
		return "airflow"
	}

	if _, ok := os.LookupEnv("TEAMCITY_VERSION"); ok {
		return "teamcity"
	}

	if _, ok := os.LookupEnv("GO_PIPELINE_NAME"); ok {
		return "gocd"
	}

	if _, ok := os.LookupEnv("SEMAPHORE"); ok {
		return "semaphore-ci"
	}

	if _, ok := os.LookupEnv("BITBUCKET_BUILD_NUMBER"); ok {
		return "bitbucket-pipelines"
	}

	if _, ok := os.LookupEnv("DRONE"); ok {
		return "drone"
	}

	return "others|local"
}
