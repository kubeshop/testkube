package testworkflowconfig

import "time"

const (
	FeatureFlagNewArchitecture = "exec"
	FeatureFlagCloudStorage    = "tw-storage"
)

type InternalConfig struct {
	Execution    ExecutionConfig    `json:"e,omitempty"`
	Workflow     WorkflowConfig     `json:"w,omitempty"`
	Resource     ResourceConfig     `json:"r,omitempty"`
	ControlPlane ControlPlaneConfig `json:"c,omitempty"`
	Worker       WorkerConfig       `json:"W,omitempty"`
}

type ExecutionConfig struct {
	Id              string            `json:"i,omitempty"`
	GroupId         string            `json:"g,omitempty"`
	Name            string            `json:"n,omitempty"`
	Number          int32             `json:"N,omitempty"`
	ScheduledAt     time.Time         `json:"s,omitempty"`
	DisableWebhooks bool              `json:"D,omitempty"`
	Tags            map[string]string `json:"t,omitempty"`
	Debug           bool              `json:"d,omitempty"`
	OrganizationId  string            `json:"o,omitempty"`
	EnvironmentId   string            `json:"e,omitempty"`
	ParentIds       string            `json:"p,omitempty"`
	PvcNames        map[string]string `json:"c,omitempty"`
}

type WorkflowConfig struct {
	Name   string            `json:"w,omitempty"`
	Labels map[string]string `json:"l,omitempty"`
}

type ControlPlaneConfig struct {
	DashboardUrl   string `json:"D,omitempty"` // TODO: Should be in different place?
	CDEventsTarget string `json:"c,omitempty"` // TODO: Should it be used by execution directly?
}

type ResourceConfig struct {
	Id       string `json:"i,omitempty"`
	RootId   string `json:"r,omitempty"`
	FsPrefix string `json:"f,omitempty"`
}

type SignatureConfig struct {
	Signature
	Children []Signature `json:"children,omitempty"`
}

type Signature struct {
	Ref      string `json:"ref,omitempty"`
	Name     string `json:"name,omitempty"`
	Category string `json:"category,omitempty"`
}

type ContainerResourceConfig struct {
	Requests ContainerResources `json:"r,omitempty"`
	Limits   ContainerResources `json:"l,omitempty"`
}

type ContainerResources struct {
	Memory string `json:"m,omitempty"`
	CPU    string `json:"c,omitempty"`
}

type WorkerConfig struct {
	Namespace             string `json:"n,omitempty"`
	DefaultRegistry       string `json:"R,omitempty"` // TODO: think if that shouldn't be Control Plane setup
	DefaultServiceAccount string `json:"s,omitempty"`
	ClusterID             string `json:"c,omitempty"`

	InitImage                         string        `json:"i,omitempty"`
	ToolkitImage                      string        `json:"t,omitempty"`
	ImageInspectorPersistenceEnabled  bool          `json:"p,omitempty"`
	ImageInspectorPersistenceCacheKey string        `json:"P,omitempty"`
	ImageInspectorPersistenceCacheTTL time.Duration `json:"T,omitempty"`

	Connection   WorkerConnectionConfig `json:"C,omitempty"`
	FeatureFlags map[string]string      `json:"f,omitempty"`
}

type WorkerConnectionConfig struct {
	Url         string `json:"C,omitempty"`
	ApiKey      string `json:"a,omitempty"`
	AgentID     string `json:"I,omitempty"`
	SkipVerify  bool   `json:"v,omitempty"`
	TlsInsecure bool   `json:"i,omitempty"`

	LocalApiUrl string `json:"A,omitempty"` // TODO: Avoid using internal API with Control Plane
}
