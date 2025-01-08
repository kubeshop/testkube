package runner

import (
	"context"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/cloud"
)

type newLogPresigner struct {
	organizationId string
	environmentId  string
	agentId        string
	grpcClient     cloud.TestKubeCloudAPIClient
	grpcApiToken   string
}

func (p *newLogPresigner) PresignSaveLog(ctx context.Context, id string, _ string) (string, error) {
	md := metadata.New(map[string]string{apiKeyMeta: p.grpcApiToken, orgIdMeta: p.organizationId, agentIdMeta: p.agentId})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	res, err := p.grpcClient.SaveExecutionLogsPresigned(metadata.NewOutgoingContext(ctx, md), &cloud.SaveExecutionLogsPresignedRequest{
		EnvironmentId: p.environmentId,
		Id:            id,
	}, opts...)
	if err != nil {
		return "", err
	}
	return res.Url, nil
}
