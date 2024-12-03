package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/log"
)

type grpcstatus interface {
	GRPCStatus() *status.Status
}

type CommandHandler func(ctx context.Context, req *cloud.CommandRequest) (*cloud.CommandResponse, error)
type CommandHandlers map[executor.Command]CommandHandler

func Handler[T any, U any](fn func(ctx context.Context, payload T) (U, error)) func(ctx context.Context, req *cloud.CommandRequest) (*cloud.CommandResponse, error) {
	return func(ctx context.Context, req *cloud.CommandRequest) (*cloud.CommandResponse, error) {
		data, _ := read[T](req.Payload)
		value, err := fn(ctx, data)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, status.Error(codes.NotFound, NewNotFoundErr("").Error())
			}
			if _, ok := err.(grpcstatus); ok {
				return nil, err
			}
			log.DefaultLogger.Errorw(fmt.Sprintf("command %s failed", req.Command), "error", err)
			return nil, status.Error(codes.Internal, err.Error())
		}
		return marshal(value)
	}
}

func read[T any](payload *structpb.Struct) (v T, err error) {
	err = cycleJSON(payload, &v)
	if err != nil {
		return v, status.Error(codes.Internal, "error unmarshalling payload")
	}
	return v, nil
}

func marshal(response any) (*cloud.CommandResponse, error) {
	jsonResponse, err := json.Marshal(response)
	commandResponse := cloud.CommandResponse{Response: jsonResponse}
	return &commandResponse, err
}

func cycleJSON(src any, tgt any) error {
	b, _ := toJSON(src)
	return fromJSON(b, tgt)
}

func toJSON(src any) (json.RawMessage, error) {
	return jsoniter.Marshal(src)
}

func fromJSON(msg json.RawMessage, tgt any) error {
	return jsoniter.Unmarshal(msg, tgt)
}

func GetLegacyRunningContext(req *cloud.ScheduleRequest) (runningContext *testkube.TestWorkflowRunningContext) {
	if req.RunningContext == nil {
		return nil
	}
	switch req.RunningContext.Type {
	case cloud.RunningContextType_UI:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.USER_TestWorkflowRunningContextActorType),
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
				Type_: common.Ptr(testkube.UI_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_CLI:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.USER_TestWorkflowRunningContextActorType),
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
				Type_: common.Ptr(testkube.CLI_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_CICD:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.USER_TestWorkflowRunningContextActorType),
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
				Type_: common.Ptr(testkube.CICD_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_CRON:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
				Type_: common.Ptr(testkube.INTERNAL_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_TESTTRIGGER:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.TESTTRIGGER_TestWorkflowRunningContextActorType),
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
				Type_: common.Ptr(testkube.INTERNAL_TestWorkflowRunningContextInterfaceType),
			},
		}
	case cloud.RunningContextType_KUBERNETESOBJECT:
		return &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Type_: common.Ptr(testkube.TESTWORKFLOWEXECUTION_TestWorkflowRunningContextActorType),
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.RunningContext.Name,
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
				Type_:         common.Ptr(testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType),
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Name:  req.ParentExecutionIds[len(req.ParentExecutionIds)-1],
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
	for i := range req.Selectors {
		if req.Selectors[i] == nil {
			return errors.New("invalid selector provided")
		}
		if req.Selectors[i].Name != "" && len(req.Selectors[i].LabelSelector) > 0 {
			return errors.New("invalid selector provided")
		}
		if req.Selectors[i].Name == "" && len(req.Selectors[i].LabelSelector) == 0 {
			return errors.New("invalid selector provided")
		}
		if req.Selectors[i].Name != "" {
			nameSelectorsCount++
		} else {
			labelSelectorsCount++
		}
	}

	// Validate if that could be Kubernetes object
	if req.KubernetesObjectName != "" && (nameSelectorsCount != 1 || labelSelectorsCount != 0) {
		return errors.New("kubernetes object can trigger only execution of a single named TestWorkflow")
	}

	return nil
}
