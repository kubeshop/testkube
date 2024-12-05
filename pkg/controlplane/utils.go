package controlplane

import (
	"context"
	"encoding/json"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/pkg/cloud"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/log"
)

type grpcstatus interface {
	GRPCStatus() *status.Status
}

type CommandHandler func(ctx context.Context, req *cloud.CommandRequest) (*cloud.CommandResponse, error)
type CommandHandlers map[cloudexecutor.Command]CommandHandler

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
