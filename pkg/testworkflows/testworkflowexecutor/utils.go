package testworkflowexecutor

import (
	"context"
	errors2 "errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secretmanager"
)

type Stream[T any] interface {
	Error() error
	Channel() <-chan T
}

type stream[T any] struct {
	errs []error
	ch   <-chan T
	mu   sync.RWMutex
}

func NewStream[T any](ch <-chan T) *stream[T] {
	return &stream[T]{ch: ch}
}

func (s *stream[T]) addError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, err)
}

func (s *stream[T]) Error() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return errors2.Join(s.errs...)
}

func (s *stream[T]) Channel() <-chan T {
	return s.ch
}

func retry(count int, delayBase time.Duration, fn func() error) (err error) {
	for i := 0; i < count; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * delayBase)
	}
	return err
}

func GetNewRunningContext(legacy *testkube.TestWorkflowRunningContext, parentExecutionIds []string) (runningContext *cloud.RunningContext, untrustedUser *cloud.UserSignature) {
	if legacy != nil {
		if legacy.Actor != nil && legacy.Actor.Type_ != nil {
			switch *legacy.Actor.Type_ {
			case testkube.CRON_TestWorkflowRunningContextActorType:
				runningContext = &cloud.RunningContext{Type: cloud.RunningContextType_CRON, Name: legacy.Actor.Name}
			case testkube.TESTTRIGGER_TestWorkflowRunningContextActorType:
				runningContext = &cloud.RunningContext{Type: cloud.RunningContextType_TESTTRIGGER, Name: legacy.Actor.Name}
			case testkube.TESTWORKFLOWEXECUTION_TestWorkflowRunningContextActorType:
				runningContext = &cloud.RunningContext{Type: cloud.RunningContextType_KUBERNETESOBJECT, Name: legacy.Actor.Name}
			case testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType:
				if len(parentExecutionIds) > 0 {
					runningContext = &cloud.RunningContext{Type: cloud.RunningContextType_EXECUTION, Name: legacy.Actor.Name, Id: parentExecutionIds[len(parentExecutionIds)-1]}
				}
			case testkube.USER_TestWorkflowRunningContextActorType:
				if legacy.Actor.Name != "" && legacy.Actor.Email != "" {
					untrustedUser = &cloud.UserSignature{Name: legacy.Actor.Name, Email: legacy.Actor.Email}
				}
			}
		}
		if runningContext == nil && legacy.Interface_ != nil && legacy.Interface_.Type_ != nil {
			switch *legacy.Interface_.Type_ {
			case testkube.CLI_TestWorkflowRunningContextInterfaceType:
				runningContext = &cloud.RunningContext{Type: cloud.RunningContextType_CLI, Name: legacy.Interface_.Name}
			case testkube.UI_TestWorkflowRunningContextInterfaceType:
				runningContext = &cloud.RunningContext{Type: cloud.RunningContextType_UI, Name: legacy.Interface_.Name}
			case testkube.CICD_TestWorkflowRunningContextInterfaceType:
				runningContext = &cloud.RunningContext{Type: cloud.RunningContextType_CICD, Name: legacy.Interface_.Name}
			}
		}
	}
	return
}

func GetLegacyRunningContext(req *cloud.ScheduleRequest) (runningContext *testkube.TestWorkflowRunningContext) {
	userActor := &testkube.TestWorkflowRunningContextActor{
		Type_: common.Ptr(testkube.USER_TestWorkflowRunningContextActorType),
	}

	if req.User != nil {
		userActor.Name = req.User.Name
		userActor.Email = req.User.Email
	}

	if req.ExecutionReference != nil {
		userActor.ExecutionReference = *req.ExecutionReference
	}

	if req.RunningContext == nil {
		if req.User == nil {
			return nil
		}

		return &testkube.TestWorkflowRunningContext{
			Actor: userActor,
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Type_: common.Ptr(testkube.API_TestWorkflowRunningContextInterfaceType),
			},
		}
	}

	switch req.RunningContext.Type {
	case cloud.RunningContextType_UI:
		return &testkube.TestWorkflowRunningContext{
			Actor: userActor,
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
				Type_: common.Ptr(testkube.UI_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_CLI:
		return &testkube.TestWorkflowRunningContext{
			Actor: userActor,
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
				Type_: common.Ptr(testkube.CLI_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_CICD:
		return &testkube.TestWorkflowRunningContext{
			Actor: userActor,
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
				Type_: common.Ptr(testkube.CICD_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_CRON:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
				Name:  req.RunningContext.Name,
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Type_: common.Ptr(testkube.INTERNAL_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_TESTTRIGGER:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.TESTTRIGGER_TestWorkflowRunningContextActorType),
				Name:  req.RunningContext.Name,
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Type_: common.Ptr(testkube.INTERNAL_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_KUBERNETESOBJECT:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.TESTWORKFLOWEXECUTION_TestWorkflowRunningContextActorType),
				Name:  req.RunningContext.Name,
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Type_: common.Ptr(testkube.INTERNAL_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_EXECUTION:
		if len(req.ParentExecutionIds) == 0 {
			break
		}
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				ExecutionId:   req.ParentExecutionIds[len(req.ParentExecutionIds)-1],
				ExecutionPath: strings.Join(req.ParentExecutionIds, "/"),
				Name:          req.RunningContext.Name,
				Type_:         common.Ptr(testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType),
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Type_: common.Ptr(testkube.INTERNAL_TestWorkflowRunningContextInterfaceType),
			},
		}
	}
	return nil
}

// TODO: Limit selectors or maximum executions to avoid huge load?
func ValidateExecutionRequest(req *cloud.ScheduleRequest) error {
	// Validate if the selectors have exclusively name or label selector
	nameSelectorsCount := 0
	labelSelectorsCount := 0
	for i := range req.Executions {
		if req.Executions[i] == nil {
			return errors.New("invalid selector provided")
		}
		if req.Executions[i].Selector.Name != "" && len(req.Executions[i].Selector.Labels) > 0 {
			return errors.New("invalid selector provided")
		}
		if req.Executions[i].Selector.Name == "" && len(req.Executions[i].Selector.Labels) == 0 {
			return errors.New("invalid selector provided")
		}
		if req.Executions[i].Selector.Name != "" {
			nameSelectorsCount++
		} else {
			labelSelectorsCount++
		}
	}

	// Validate if that could be Kubernetes object
	if req.KubernetesObjectName != "" && (nameSelectorsCount != 1 || labelSelectorsCount != 0) {
		return errors.New("kubernetes object can trigger only execution of a single named TestWorkflow")
	}

	// Validate if that could be Resolved workflow object
	if len(req.ResolvedWorkflow) != 0 && (nameSelectorsCount != 1 || labelSelectorsCount != 0) {
		return errors.New("resolved workflow can trigger only execution of a single named TestWorkflow")
	}

	return nil
}

func ValidateExecutionNameDuplicates(intermediate []*IntermediateExecution) error {
	type namePair struct {
		Workflow  string
		Execution string
	}
	localDuplicatesCheck := make(map[namePair]struct{})
	for i := range intermediate {
		if intermediate[i].Name() == "" {
			continue
		}
		key := namePair{Workflow: intermediate[i].WorkflowName(), Execution: intermediate[i].Name()}
		if _, ok := localDuplicatesCheck[key]; ok {
			return fmt.Errorf("duplicated execution name: '%s' for workflow '%s'", intermediate[i].Name(), intermediate[i].WorkflowName())
		}
		localDuplicatesCheck[key] = struct{}{}
	}
	return nil
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mongo.ErrNoDocuments) || k8serrors.IsNotFound(err) || errors.Is(err, secretmanager.ErrNotFound) {
		return true
	}
	if e, ok := status.FromError(err); ok {
		return e.Code() == codes.NotFound
	}
	return false
}

func ValidateExecutionNameRemoteDuplicate(ctx context.Context, resultsRepository testworkflow.Repository, intermediate *IntermediateExecution) error {
	next, err := resultsRepository.GetByNameAndTestWorkflow(ctx, intermediate.Name(), intermediate.WorkflowName())
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to verify unique name: '%s' in '%s' workflow", intermediate.Name(), intermediate.WorkflowName())
	}
	if next.Name == intermediate.Name() {
		return fmt.Errorf("execution name already exists: '%s' for workflow '%s'", intermediate.Name(), intermediate.WorkflowName())
	}
	return nil
}
