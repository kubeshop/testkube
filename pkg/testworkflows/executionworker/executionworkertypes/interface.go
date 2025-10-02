package executionworkertypes

import (
	"context"
	"time"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/utils"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type ServiceConfig struct {
	RestartPolicy  string
	ReadinessProbe *testkube.Probe
}

// Runtime contains runtime overrides for test workflow execution
type Runtime struct {
	Variables map[string]string
}

// TODO: Consider some context data
type ExecuteRequest struct {
	Token       string
	ResourceId  string // defaults to execution ID
	GroupId     string
	Workflow    testworkflowsv1.TestWorkflow // TODO: Use OpenAPI object
	Secrets     map[string]map[string]string
	ScheduledAt *time.Time
	Runtime     *Runtime // Runtime configuration overrides

	Execution           testworkflowconfig.ExecutionConfig
	ControlPlane        testworkflowconfig.ControlPlaneConfig // TODO: Think if it's required
	ArtifactsPathPrefix string
}

type ServiceRequest struct {
	Token          string
	ResourceId     string
	GroupId        string
	Workflow       testworkflowsv1.TestWorkflow // TODO: Use OpenAPI object
	Secrets        map[string]map[string]string
	ScheduledAt    *time.Time
	ReadinessProbe *testkube.Probe
	RestartPolicy  string
	Runtime        *Runtime // Runtime configuration overrides

	Execution           testworkflowconfig.ExecutionConfig
	ControlPlane        testworkflowconfig.ControlPlaneConfig // TODO: Think if it's required
	ArtifactsPathPrefix string
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
	// ScheduledAt informs about scheduled time.
	ScheduledAt time.Time
	// Namespace where it has been scheduled.
	Namespace string
	// Redundant says if that execution was already running.
	Redundant bool
}

type ServiceResult struct {
	// Signature for the deployed resource.
	Signature []testkube.TestWorkflowSignature
	// ScheduledAt informs about scheduled time.
	ScheduledAt time.Time
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
	// Root filters to only root or non-root resources.
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

type GetOptions struct {
	// Hints to help to find item faster and to provide more accurate data.
	Hints Hints
}

type ControlOptions struct {
	// Namespace where it has been deployed.
	Namespace string
}

type DestroyOptions struct {
	// Namespace where it has been deployed.
	Namespace string
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

//go:generate mockgen -destination=./mock_worker.go -package=executionworkertypes "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes" Worker
type Worker interface {
	// Execute deploys the resources in the cluster.
	Execute(ctx context.Context, request ExecuteRequest) (*ExecuteResult, error)

	// Service deploys the resources for a new service in the cluster.
	Service(ctx context.Context, request ServiceRequest) (*ServiceResult, error)

	// Notifications stream all the notifications from the resource.
	Notifications(ctx context.Context, id string, options NotificationsOptions) NotificationsWatcher

	// StatusNotifications stream lightweight status information.
	StatusNotifications(ctx context.Context, id string, options StatusNotificationsOptions) StatusNotificationsWatcher

	// Logs converts all the important notifications (except i.e. output) from the resource into plain logs.
	Logs(ctx context.Context, id string, options LogsOptions) utils.LogsReader

	// Get tries to build the latest precise result from the resource execution.
	Get(ctx context.Context, id string, options GetOptions) (*GetResult, error)

	// Summary gets fast summary about the selected resource.
	Summary(ctx context.Context, id string, options GetOptions) (*SummaryResult, error)

	// Finished is a fast method to check if the resource execution has been already finished.
	Finished(ctx context.Context, id string, options GetOptions) (bool, error)

	// List lists all the currently deployed resources matching the criteria.
	List(ctx context.Context, options ListOptions) ([]ListResultItem, error)

	// Abort may either destroy or just stop the selected resource (so the data can be still accessible)
	Abort(ctx context.Context, id string, options DestroyOptions) error

	// Cancel sends cancel request to the selected resource.
	Cancel(ctx context.Context, id string, options DestroyOptions) error

	// Destroy gets rid of all the data for the selected resource.
	Destroy(ctx context.Context, id string, options DestroyOptions) error

	// DestroyGroup gets rid of all the data for the selected resources.
	DestroyGroup(ctx context.Context, groupId string, options DestroyOptions) error

	// Pause sends pause request to the selected resource.
	Pause(ctx context.Context, id string, options ControlOptions) error

	// Resume sends resuming request to the selected resource.
	Resume(ctx context.Context, id string, options ControlOptions) error

	// ResumeMany tries to resume multiple resources at once.
	ResumeMany(ctx context.Context, ids []string, options ControlOptions) (errs []IdentifiableError)
}
