package executionworker

import (
	"context"
	"time"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type NamespaceConfig struct {
	DefaultServiceAccountName string
}

type ClusterConfig struct {
	Id               string
	DefaultNamespace string
	DefaultRegistry  string
	Namespaces       map[string]NamespaceConfig
}

type ImageInspectorConfig struct {
	CacheEnabled bool
	CacheKey     string
	CacheTTL     time.Duration
}

type ServiceConfig struct {
	RestartPolicy  string
	ReadinessProbe *testkube.Probe
}

type Config struct {
	Cluster        ClusterConfig
	ImageInspector ImageInspectorConfig
	Connection     testworkflowconfig.WorkerConnectionConfig
}

// TODO: Consider some context data?
// TODO: Support sub-resources (`parallel` and `services`)?
type ExecuteRequest struct {
	Execution testworkflowconfig.ExecutionConfig
	Secrets   map[string]map[string]string
	Workflow  testworkflowsv1.TestWorkflow // TODO: Use OpenAPI object

	ControlPlane testworkflowconfig.ControlPlaneConfig // TODO: Think if it's required

	// TODO: Think if it should be wrapped differently
	ResourceId string
	GroupId    string // TODO: likely it should be part of the resource config
	FsPrefix   string
	Service    *ServiceConfig // TODO: Think if service should not be executed by a different command
}

type Hints struct {
	// Namespace to search for firstly.
	Namespace string
	// ScheduledAt time to align with the execution.
	ScheduledAt *time.Time // TODO: Consider no pointer
	// Signature to align with the execution.
	Signature []testkube.TestWorkflowSignature
}

type ExecuteResult struct {
	// Signature for the deployed resource.
	Signature []testkube.TestWorkflowSignature
	// Namespace where it has been scheduled.
	Namespace string
}

type SignatureResult struct {
	// Signature for the selected resource.
	Signature []testkube.TestWorkflowSignature
}

type GetResult struct {
	// Execution details
	Execution testworkflowconfig.ExecutionConfig
	// Workflow basic metadata.
	Workflow testworkflowconfig.WorkflowConfig
	// Resource details.
	Resource testworkflowconfig.ResourceConfig
	// Signature for the resource.
	Signature []testkube.TestWorkflowSignature
	// Result keeps the latest recognized status of the execution.
	Result testkube.TestWorkflowResult
	// Namespace where it has been deployed to.
	Namespace string
}

type ListOptions struct {
	// RootId filters the root ID the search for the deployed resources.
	RootId string
	// OrganizationId filters by organization ID tied to the execution.
	OrganizationId string
	// EnvironmentId filters by environment ID tied to the execution.
	EnvironmentId string
	// GroupId filters by group ID of the resources.
	GroupId string
	// Root filters to only root or non-root resources. TODO: Consider root-only as a default?
	Root *bool
	// Finished filters based on the execution being finished or still running.
	Finished *bool
	// Namespaces to search in specific namespaces.
	Namespaces []string
}

type ListResultItem struct {
	// Execution details.
	Execution testworkflowconfig.ExecutionConfig
	// Workflow basic metadata.
	Workflow testworkflowconfig.WorkflowConfig
	// Resource details.
	Resource testworkflowconfig.ResourceConfig
	// Namespace where it has been deployed.
	Namespace string
}

type SummaryResult struct {
	// Execution details
	Execution testworkflowconfig.ExecutionConfig
	// Workflow basic metadata.
	Workflow testworkflowconfig.WorkflowConfig
	// Resource details.
	Resource testworkflowconfig.ResourceConfig
	// Signature for the resource.
	Signature []testkube.TestWorkflowSignature
	// EstimatedResult keeps the best estimated status of the execution.
	// It may be not precise, i.e. timestamps may be not accurate, or more steps may be finished already.
	// The statuses of finished steps and the workflow itself are guaranteed to be valid though.
	EstimatedResult testkube.TestWorkflowResult
	// Namespace where it has been deployed.
	Namespace string
}

type NotificationsOptions struct {
	// Hints to help to find item faster and to provide more accurate data.
	Hints Hints
	// NoFollow gives a hint to ignore following the further actions.
	NoFollow bool
}

type LogsOptions struct {
	// Hints to help to find item faster and to provide more accurate data.
	Hints Hints
	// NoFollow gives a hint to ignore following the further actions.
	NoFollow bool
}

type StatusNotificationsOptions struct {
	// Hints to help to find item faster and to provide more accurate data.
	Hints Hints
	// NoFollow gives a hint to ignore following the further actions.
	NoFollow bool
}

type StatusNotification struct {
	// NodeName is provided when the Pod is scheduled on some node.
	NodeName string
	// PodIp is internal IP of the Pod.
	PodIp string
	// Ready states for container readiness if expected (services).
	Ready bool
	// Ref provides information about current step reference.
	Ref string
	// Result stores the latest result change.
	Result *testkube.TestWorkflowResult
}

type IdentifiableError struct {
	// Id is an ID of the resource associated to the error.
	Id string
	// Error is the error that happened.
	Error error
}

//go:generate mockgen -destination=./mock_worker.go -package=executionworker "github.com/kubeshop/testkube/pkg/testworkflows/executionworker" Worker
type Worker interface {
	// Execute deploys the resources in the cluster.
	Execute(ctx context.Context, request ExecuteRequest) (*ExecuteResult, error)

	// Notifications stream all the notifications from the resource.
	Notifications(ctx context.Context, id string, options NotificationsOptions) NotificationsWatcher

	// StatusNotifications stream lightweight status information.
	StatusNotifications(ctx context.Context, id string, options StatusNotificationsOptions) StatusNotificationsWatcher

	// Logs converts all the important notifications (except i.e. output) from the resource into plain logs.
	Logs(ctx context.Context, id string, options LogsOptions) LogsReader

	// Get tries to build the latest precise result from the resource execution.
	Get(ctx context.Context, namespace, id string) (*GetResult, error)

	// Summary gets fast summary about the selected resource.
	Summary(ctx context.Context, namespace, id string) (*SummaryResult, error)

	// Finished is a fast method to check if the resource execution has been already finished.
	Finished(ctx context.Context, namespace, id string) (bool, error)

	// ListIds lists all the IDs of currently deployed resources matching the criteria.
	ListIds(ctx context.Context, options ListOptions) ([]string, error)

	// List lists all the currently deployed resources matching the criteria.
	List(ctx context.Context, options ListOptions) ([]ListResultItem, error)

	// Destroy gets rid of all the data for the selected resource.
	Destroy(ctx context.Context, namespace, id string) error

	// DestroyGroup gets rid of all the data for the selected resources.
	DestroyGroup(ctx context.Context, namespace, groupId string) error

	// Pause sends pause request to the selected resource.
	Pause(ctx context.Context, namespace, id string) error

	// Resume sends resuming request to the selected resource.
	Resume(ctx context.Context, namespace, id string) error

	// ResumeMany tries to resume multiple resources at once.
	ResumeMany(ctx context.Context, ids []string) (errs []IdentifiableError)
}
