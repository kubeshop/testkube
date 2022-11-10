package telemetry

import (
	"runtime"
	"strings"

	"github.com/kubeshop/testkube/pkg/utils/text"
)

type Params struct {
	EventCount       int64  `json:"event_count,omitempty"`
	EventCategory    string `json:"event_category,omitempty"`
	AppVersion       string `json:"app_version,omitempty"`
	AppName          string `json:"app_name,omitempty"`
	CustomDimensions string `json:"custom_dimensions,omitempty"`
	DataSource       string `json:"data_source,omitempty"`
	Host             string `json:"host,omitempty"`
	MachineID        string `json:"machine_id,omitempty"`
	ClusterID        string `json:"cluster_id,omitempty"`
	OperatingSystem  string `json:"operating_system,omitempty"`
	Architecture     string `json:"architecture,omitempty"`
	TestType         string `json:"test_type,omitempty"`
	DurationMs       int32  `json:"duration_ms,omitempty"`
	Status           string `json:"status,omitempty"`
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

type CreateParams struct {
	AppVersion string
	DataSource string
	Host       string
	ClusterID  string
	TestType   string
}

type RunParams struct {
	AppVersion string
	DataSource string
	Host       string
	ClusterID  string
	TestType   string
	DurationMs int32
	Status     string
}

func NewCLIPayload(id, name, version, category string) Payload {
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
				},
			}},
	}
}

func NewAPIPayload(clusterId, name, version, host string) Payload {
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
				},
			}},
	}
}

func NewCreatePayload(name string, params CreateParams) Payload {
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
					AppName:         "testkube-operator",
					DataSource:      params.DataSource,
					Host:            AnonymizeHost(params.Host),
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					MachineID:       GetMachineID(),
					ClusterID:       params.ClusterID,
					TestType:        params.TestType,
				},
			}},
	}
}

func NewRunPayload(name string, params RunParams) Payload {
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
					DataSource:      params.DataSource,
					Host:            AnonymizeHost(params.Host),
					OperatingSystem: runtime.GOOS,
					Architecture:    runtime.GOARCH,
					MachineID:       GetMachineID(),
					ClusterID:       params.ClusterID,
					TestType:        params.TestType,
					DurationMs:      params.DurationMs,
					Status:          params.Status,
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
	} else if strings.Contains(host, "localhost:8088") {
		return APIHostLocal
	}

	return APIHostExternal
}
