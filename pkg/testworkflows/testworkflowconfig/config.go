package testworkflowconfig

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	FeatureFlagCloudStorage = "tw-storage"
)

type InternalConfig struct {
	Execution    ExecutionConfig    `json:"e,omitempty"`
	Workflow     WorkflowConfig     `json:"w,omitempty"`
	Resource     ResourceConfig     `json:"r,omitempty"`
	ControlPlane ControlPlaneConfig `json:"c,omitempty"`
	Worker       WorkerConfig       `json:"W,omitempty"`
}

type ExecutionConfig struct {
	Id               string                               `json:"i,omitempty"`
	GroupId          string                               `json:"g,omitempty"`
	Name             string                               `json:"n,omitempty"`
	Number           int32                                `json:"N,omitempty"`
	ScheduledAt      time.Time                            `json:"s,omitempty"`
	DisableWebhooks  bool                                 `json:"D,omitempty"`
	Tags             map[string]string                    `json:"t,omitempty"`
	Debug            bool                                 `json:"d,omitempty"`
	OrganizationId   string                               `json:"o,omitempty"`
	OrganizationSlug string                               `json:"O,omitempty"`
	EnvironmentId    string                               `json:"e,omitempty"`
	EnvironmentSlug  string                               `json:"E,omitempty"`
	ParentIds        string                               `json:"p,omitempty"`
	PvcNames         map[string]string                    `json:"c,omitempty"`
	GlobalEnv        []testworkflowsv1.EnvVar             `json:"G,omitempty"`
	SecretMountPaths map[string][]string                  `json:"S,omitempty"`
	RunningContext   *testkube.TestWorkflowRunningContext `json:"R,omitempty"`
	Rerun            *RerunConfig                         `json:"U,omitempty"`
}

// RerunConfig narrows an execution to specific test cases.
//
// Only the reference travels: the selection itself is resolved in the pod
// against the report the referenced execution produced. A suite of ten thousand
// test cases would not survive being carried as a list through scheduling, and
// the pod is where the report already is.
//
// It has an effect only on a step declaring `testCases.select`, since Testkube
// does not know any runner's filter flag - the workflow has to say how selected
// names reach the tool.
type RerunConfig struct {
	// ExecutionId is where the previous results are read from.
	ExecutionId string `json:"i,omitempty"`
	// OnlyFailed restricts the selection to test cases that did not pass.
	OnlyFailed bool `json:"f,omitempty"`
	// TestCases is an explicit selection, instead of or alongside ExecutionId.
	TestCases []string `json:"c,omitempty"`
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
	Namespace             string   `json:"n,omitempty"`
	DefaultRegistry       string   `json:"R,omitempty"` // TODO: think if that shouldn't be Control Plane setup
	InsecureRegistries    []string `json:"x,omitempty"`
	DefaultServiceAccount string   `json:"s,omitempty"`
	ClusterID             string   `json:"c,omitempty"`
	RunnerID              string   `json:"r,omitempty"`

	InitImage                         string        `json:"i,omitempty"`
	ToolkitImage                      string        `json:"t,omitempty"`
	ImageInspectorPersistenceEnabled  bool          `json:"p,omitempty"`
	ImageInspectorPersistenceCacheKey string        `json:"P,omitempty"`
	ImageInspectorPersistenceCacheTTL time.Duration `json:"T,omitempty"`

	Connection             WorkerConnectionConfig `json:"C,omitempty"`
	FeatureFlags           map[string]string      `json:"f,omitempty"`
	CommonEnvVariables     []corev1.EnvVar        `json:"e,omitempty"`
	AllowLowSecurityFields bool                   `json:"a,omitempty"`
	DisableResourceMetrics bool                   `json:"m,omitempty"`
	EmptyDirSizeLimit      string                 `json:"z,omitempty"`

	DefaultImagePullPolicy string                  `json:"Y,omitempty"`
	DefaultRunnerResources ContainerResourceConfig `json:"D,omitempty"`
}

type WorkerConnectionConfig struct {
	Url         string `json:"C,omitempty"`
	ApiKey      string `json:"a,omitempty"`
	AgentID     string `json:"I,omitempty"`
	SkipVerify  bool   `json:"v,omitempty"`
	TlsInsecure bool   `json:"i,omitempty"`

	LocalApiUrl string `json:"A,omitempty"` // TODO: Avoid using internal API with Control Plane
}
