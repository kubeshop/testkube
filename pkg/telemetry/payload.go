package telemetry

import (
	"runtime"
	"strings"

	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

const runContextAgent = "agent"

type Params struct {
	EventCount                 int64      `json:"event_count,omitempty"`
	EventCategory              string     `json:"event_category,omitempty"`
	AppVersion                 string     `json:"app_version,omitempty"`
	AppName                    string     `json:"app_name,omitempty"`
	CustomDimensions           string     `json:"custom_dimensions,omitempty"`
	DataSource                 string     `json:"data_source,omitempty"`
	Host                       string     `json:"host,omitempty"`
	MachineID                  string     `json:"machine_id,omitempty"`
	ClusterID                  string     `json:"cluster_id,omitempty"`
	OperatingSystem            string     `json:"operating_system,omitempty"`
	Architecture               string     `json:"architecture,omitempty"`
	TestType                   string     `json:"test_type,omitempty"`
	DurationMs                 int32      `json:"duration_ms,omitempty"`
	Status                     string     `json:"status,omitempty"`
	TestSource                 string     `json:"test_source,omitempty"`
	TestSuiteSteps             int32      `json:"test_suite_steps,omitempty"`
	Context                    RunContext `json:"context,omitempty"`
	ClusterType                string     `json:"cluster_type,omitempty"`
	CliContext                 string     `json:"cli_context,omitempty"`
	Error                      string     `json:"error,omitempty"`
	ErrorType                  string     `json:"error_type,omitempty"`
	ErrorStackTrace            string     `json:"error_stacktrace,omitempty"`
	TestWorkflowSteps          int32      `json:"test_workflow_steps,omitempty"`
	TestWorkflowTemplateUsed   bool       `json:"test_workflow_template_used,omitempty"`
	TestWorkflowImage          string     `json:"test_workflow_image,omitempty"`
	TestWorkflowArtifactUsed   bool       `json:"test_workflow_artifact_used,omitempty"`
	TestWorkflowKubeshopGitURI bool       `json:"test_workflow_kubeshop_git_uri,omitempty"`
	License                    string     `json:"license,omitempty"`
}

type Event struct {
	Name   string `json:"name"`
	Params Params `json:"params,omitempty"`
}

type Payload struct {
	UserID   string  `json:"user_id,omitempty"`
	ClientID string  `json:"client_id,omitempty"`
	Events   []Event `json:"events,omitempty"`
}

// CreateParams contains Test or Test suite creation parameters
type CreateParams struct {
	AppVersion     string
	DataSource     string
	Host           string
	ClusterID      string
	TestType       string
	TestSource     string
	TestSuiteSteps int32
}

// RunParams contains Test or Test suite run parameters
type RunParams struct {
	AppVersion string
	DataSource string
	Host       string
	ClusterID  string
	TestType   string
	DurationMs int32
	Status     string
}

type RunContext struct {
	Type           string
	OrganizationId string
	EnvironmentId  string
}

type WorkflowParams struct {
	TestWorkflowSteps          int32
	TestWorkflowTemplateUsed   bool
	TestWorkflowImage          string
	TestWorkflowArtifactUsed   bool
	TestWorkflowKubeshopGitURI bool
}

type CreateWorkflowParams struct {
	CreateParams
	WorkflowParams
}

type RunWorkflowParams struct {
	RunParams
	WorkflowParams
}

func NewCLIPayload(context RunContext, id, name, version, category, clusterType string) Payload {
	return Payload{
		ClientID: id,
		UserID:   id,
		Events: []Event{
			{
				Name: text.GAEventName(name),
				Params: Params{
					EventCount:      1,
					EventCategory:   category,
					AppVersion:      version,
					AppName:         "kubectl-testkube",
					MachineID:       GetMachineID(),
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					Context:         context,
					ClusterType:     clusterType,
					CliContext:      GetCliRunContext(),
				},
			}},
	}
}

func NewCLIWithLicensePayload(context RunContext, id, name, version, category, clusterType, license string) Payload {
	return Payload{
		ClientID: id,
		UserID:   id,
		Events: []Event{
			{
				Name: text.GAEventName(name),
				Params: Params{
					EventCount:      1,
					EventCategory:   category,
					AppVersion:      version,
					AppName:         "kubectl-testkube",
					MachineID:       GetMachineID(),
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					Context:         context,
					ClusterType:     clusterType,
					CliContext:      GetCliRunContext(),
					License:         license,
				},
			}},
	}
}

func NewAPIPayload(clusterId, name, version, host, clusterType string) Payload {
	return Payload{
		ClientID: clusterId,
		UserID:   clusterId,
		Events: []Event{
			{
				Name: text.GAEventName(name),
				Params: Params{
					EventCount:      1,
					EventCategory:   "api",
					AppVersion:      version,
					AppName:         "testkube-api-server",
					Host:            AnonymizeHost(host),
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					MachineID:       GetMachineID(),
					ClusterID:       clusterId,
					ClusterType:     clusterType,
					Context:         getAgentContext(),
				},
			}},
	}
}

// NewCreatePayload prepares payload for Test or Test suite creation
func NewCreatePayload(name, clusterType string, params CreateParams) Payload {
	return Payload{
		ClientID: params.ClusterID,
		UserID:   params.ClusterID,
		Events: []Event{
			{
				Name: text.GAEventName(name),
				Params: Params{
					EventCount:      1,
					EventCategory:   "api",
					AppVersion:      params.AppVersion,
					AppName:         "testkube-api-server",
					Host:            AnonymizeHost(params.Host),
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					MachineID:       GetMachineID(),
					ClusterID:       params.ClusterID,
					DataSource:      params.DataSource,
					TestType:        params.TestType,
					TestSource:      params.TestSource,
					TestSuiteSteps:  params.TestSuiteSteps,
					ClusterType:     clusterType,
					Context:         getAgentContext(),
				},
			}},
	}
}

// NewRunPayload prepares payload for Test or Test suite execution
func NewRunPayload(name, clusterType string, params RunParams) Payload {
	return Payload{
		ClientID: params.ClusterID,
		UserID:   params.ClusterID,
		Events: []Event{
			{
				Name: text.GAEventName(name),
				Params: Params{
					EventCount:      1,
					EventCategory:   "api",
					AppVersion:      params.AppVersion,
					AppName:         "testkube-api-server",
					Host:            AnonymizeHost(params.Host),
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					MachineID:       GetMachineID(),
					ClusterID:       params.ClusterID,
					DataSource:      params.DataSource,
					TestType:        params.TestType,
					DurationMs:      params.DurationMs,
					Status:          params.Status,
					ClusterType:     clusterType,
					Context:         getAgentContext(),
				},
			}},
	}
}

// NewCreateWorkflowPayload prepares payload for Test workflow creation
func NewCreateWorkflowPayload(name, clusterType string, params CreateWorkflowParams) Payload {
	return Payload{
		ClientID: params.ClusterID,
		UserID:   params.ClusterID,
		Events: []Event{
			{
				Name: text.GAEventName(name),
				Params: Params{
					EventCount:                 1,
					EventCategory:              "api",
					AppVersion:                 params.AppVersion,
					AppName:                    "testkube-api-server",
					Host:                       AnonymizeHost(params.Host),
					OperatingSystem:            runtime.GOOS,
					Architecture:               runtime.GOARCH,
					MachineID:                  GetMachineID(),
					ClusterID:                  params.ClusterID,
					DataSource:                 params.DataSource,
					TestType:                   params.TestType,
					TestSource:                 params.TestSource,
					TestSuiteSteps:             params.TestSuiteSteps,
					ClusterType:                clusterType,
					Context:                    getAgentContext(),
					TestWorkflowSteps:          params.TestWorkflowSteps,
					TestWorkflowTemplateUsed:   params.TestWorkflowTemplateUsed,
					TestWorkflowImage:          params.TestWorkflowImage,
					TestWorkflowArtifactUsed:   params.TestWorkflowArtifactUsed,
					TestWorkflowKubeshopGitURI: params.TestWorkflowKubeshopGitURI,
				},
			}},
	}
}

// NewRunWorkflowPayload prepares payload for Test workflow execution
func NewRunWorkflowPayload(name, clusterType string, params RunWorkflowParams) Payload {
	return Payload{
		ClientID: params.ClusterID,
		UserID:   params.ClusterID,
		Events: []Event{
			{
				Name: text.GAEventName(name),
				Params: Params{
					EventCount:                 1,
					EventCategory:              "api",
					AppVersion:                 params.AppVersion,
					AppName:                    "testkube-api-server",
					Host:                       AnonymizeHost(params.Host),
					OperatingSystem:            runtime.GOOS,
					Architecture:               runtime.GOARCH,
					MachineID:                  GetMachineID(),
					ClusterID:                  params.ClusterID,
					DataSource:                 params.DataSource,
					TestType:                   params.TestType,
					DurationMs:                 params.DurationMs,
					Status:                     params.Status,
					ClusterType:                clusterType,
					Context:                    getAgentContext(),
					TestWorkflowSteps:          params.TestWorkflowSteps,
					TestWorkflowTemplateUsed:   params.TestWorkflowTemplateUsed,
					TestWorkflowImage:          params.TestWorkflowImage,
					TestWorkflowArtifactUsed:   params.TestWorkflowArtifactUsed,
					TestWorkflowKubeshopGitURI: params.TestWorkflowKubeshopGitURI,
				},
			}},
	}
}

const (
	APIHostLocal            = "local"
	APIHostExternal         = "external"
	APIHostTestkubeInternal = "testkube-internal"
)

func AnonymizeHost(host string) string {
	if strings.Contains(host, "testkube.io") {
		return APIHostTestkubeInternal
	} else if strings.Contains(host, "localhost:") {
		return APIHostLocal
	}

	return APIHostExternal
}

func getAgentContext() RunContext {
	orgID := utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_ORG_ID", "TESTKUBE_CLOUD_ORG_ID", "")
	envID := utils.GetEnvVarWithDeprecation("TESTKUBE_PRO_ENV_ID", "TESTKUBE_CLOUD_ENV_ID", "")

	if orgID == "" || envID == "" {
		return RunContext{}
	}
	return RunContext{
		Type:           runContextAgent,
		EnvironmentId:  envID,
		OrganizationId: orgID,
	}
}
