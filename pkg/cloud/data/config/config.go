package config

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/repository/config"
)

var _ config.Repository = (*CloudRepository)(nil)

type CloudRepository struct {
	executor executor.Executor
}

func NewCloudResultRepository(cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudRepository {
	return &CloudRepository{executor: executor.NewCloudGRPCExecutor(cloudClient, grpcConn, apiKey)}
}

func (r *CloudRepository) GetUniqueClusterId(ctx context.Context) (string, error) {
	req := GetUniqueClusterIdRequest{}
	response, err := r.executor.Execute(ctx, CmdConfigGetUniqueClusterId, req)
	if err != nil {
		return "", err
	}
	var commandResponse GetUniqueClusterIdResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return "", err
	}
	return commandResponse.ClusterID, nil
}

func (r *CloudRepository) GetTelemetryEnabled(ctx context.Context) (ok bool, err error) {
	req := GetTelemetryEnabledRequest{}
	response, err := r.executor.Execute(ctx, CmdConfigGetTelemetryEnabled, req)
	if err != nil {
		return false, err
	}
	var commandResponse GetTelemetryEnabledResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return false, err
	}
	return commandResponse.Enabled, nil
}

func (r *CloudRepository) Get(ctx context.Context) (testkube.Config, error) {
	req := GetRequest{}
	response, err := r.executor.Execute(ctx, CmdConfigGet, req)
	if err != nil {
		return testkube.Config{}, err
	}
	var commandResponse GetResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.Config{}, err
	}
	return commandResponse.Config, nil
}

func (r *CloudRepository) Upsert(ctx context.Context, config testkube.Config) (testkube.Config, error) {
	req := UpsertRequest{Config: config}
	response, err := r.executor.Execute(ctx, CmdConfigUpsert, req)
	if err != nil {
		return testkube.Config{}, err
	}
	var commandResponse UpsertResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.Config{}, err
	}
	return commandResponse.Config, nil
}
