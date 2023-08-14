package telemetry

import (
	"os"
	"runtime"
	"strings"

	"github.com/kubeshop/testkube/pkg/utils/text"
)

const runContextAgent = "agent"

type Params struct {
	EventCount       int64      `json:"event_count,omitempty"`
	EventCategory    string     `json:"event_category,omitempty"`
	AppVersion       string     `json:"app_version,omitempty"`
	AppName          string     `json:"app_name,omitempty"`
	CustomDimensions string     `json:"custom_dimensions,omitempty"`
	DataSource       string     `json:"data_source,omitempty"`
	Host             string     `json:"host,omitempty"`
	MachineID        string     `json:"machine_id,omitempty"`
	ClusterID        string     `json:"cluster_id,omitempty"`
	OperatingSystem  string     `json:"operating_system,omitempty"`
	Architecture     string     `json:"architecture,omitempty"`
	TestType         string     `json:"test_type,omitempty"`
	DurationMs       int32      `json:"duration_ms,omitempty"`
	Status           string     `json:"status,omitempty"`
	TestSource       string     `json:"test_source,omitempty"`
	TestSuiteSteps   int32      `json:"test_suite_steps,omitempty"`
	Context          RunContext `json:"context,omitempty"`
	ClusterType      string     `json:"cluster_type,omitempty"`
	Error            string     `json:"error,omitempty"`
	ErrorType        string     `json:"error_type,omitempty"`
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

func NewCLIPayload(context RunContext, id, name, version, category, clusterType string) Payload {
	machineID := GetMachineID()
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
					MachineID:       machineID,
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					Context:         context,
					ClusterType:     clusterType,
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
	orgID := os.Getenv("TESTKUBE_CLOUD_ORG_ID")
	envID := os.Getenv("TESTKUBE_CLOUD_ENV_ID")

	if orgID == "" || envID == "" {
		return RunContext{}
	}
	return RunContext{
		Type:           runContextAgent,
		EnvironmentId:  envID,
		OrganizationId: orgID,
	}
}
