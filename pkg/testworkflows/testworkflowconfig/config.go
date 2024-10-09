package testworkflowconfig

import "time"

type InternalConfig struct {
	Execution    ExecutionConfig    `json:"e,omitempty"`
	Workflow     WorkflowConfig     `json:"w,omitempty"`
	Resource     ResourceConfig     `json:"r,omitempty"`
	ControlPlane ControlPlaneConfig `json:"c,omitempty"`
	Runtime      RuntimeConfig      `json:"R,omitempty"`
}

type ExecutionConfig struct {
	Id              string            `json:"i,omitempty"`
	Name            string            `json:"n,omitempty"`
	Number          int32             `json:"N,omitempty"`
	ScheduledAt     time.Time         `json:"s,omitempty"`
	DisableWebhooks bool              `json:"D,omitempty"`
	Tags            map[string]string `json:"t,omitempty"`
	Debug           bool              `json:"d,omitempty"`
	OrganizationId  string            `json:"o,omitempty"`
	EnvironmentId   string            `json:"e,omitempty"`
}

type WorkflowConfig struct {
	Name   string            `json:"w,omitempty"`
	Labels map[string]string `json:"l,omitempty"`
}

type ResourceConfig struct {
	Id       string `json:"i,omitempty"`
	RootId   string `json:"r,omitempty"`
	FsPrefix string `json:"f,omitempty"`
}

type ControlPlaneConfig struct {
	DashboardUrl   string `json:"D,omitempty"` // TODO: Should be in different place?
	CDEventsTarget string `json:"c,omitempty"`
}

type RuntimeConfig struct {
	Namespace             string `json:"n,omitempty"`
	DefaultRegistry       string `json:"R,omitempty"`
	DefaultServiceAccount string `json:"s,omitempty"`
	ClusterID             string `json:"c,omitempty"`

	InitImage                         string        `json:"i,omitempty"`
	ToolkitImage                      string        `json:"t,omitempty"`
	ImageInspectorPersistenceEnabled  bool          `json:"p,omitempty"`
	ImageInspectorPersistenceCacheKey string        `json:"P,omitempty"`
	ImageInspectorPersistenceCacheTTL time.Duration `json:"T,omitempty"`

	Connection RuntimeConnectionConfig `json:"C,omitempty"`
}

type RuntimeConnectionConfig struct {
	Url         string `json:"C,omitempty"`
	ApiKey      string `json:"a,omitempty"`
	SkipVerify  bool   `json:"v,omitempty"`
	TlsInsecure bool   `json:"i,omitempty"`

	LocalApiUrl   string              `json:"A,omitempty"` // TODO: Avoid using internal API with Control Plane
	ObjectStorage ObjectStorageConfig `json:"O,omitempty"` // TODO: Avoid using Object Storage only directly
}

// TODO: Avoid using Object Storage directly
type ObjectStorageConfig struct {
	Endpoint        string `json:"e,omitempty"`
	AccessKeyID     string `json:"a,omitempty"`
	SecretAccessKey string `json:"s,omitempty"`
	Region          string `json:"r,omitempty"`
	Token           string `json:"t,omitempty"`
	Bucket          string `json:"b,omitempty"`
	Ssl             bool   `json:"S,omitempty"`
	SkipVerify      bool   `json:"v,omitempty"`
	CertFile        string `json:"c,omitempty"`
	KeyFile         string `json:"k,omitempty"`
	CAFile          string `json:"C,omitempty"`
}
