// Package grpc is the Agent-side gRPC client for AgentInventoryService: RPCs
// the Agent uses to push cluster-environment snapshots to the Control Plane.
// Mirrors internal/sync/grpc shape and conventions.
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/grpcutils"
	inventoryv1 "github.com/kubeshop/testkube/pkg/proto/testkube/inventory/v1"
)

const defaultCallTimeout = 30 * time.Second

type Client struct {
	OrganizationId string

	client      inventoryv1.AgentInventoryServiceClient
	logger      *zap.SugaredLogger
	callOpts    []grpc.CallOption
	callTimeout time.Duration
}

func NewClient(conn grpc.ClientConnInterface, logger *zap.SugaredLogger, apiToken, organizationId string, tlsEnabled bool) Client {
	c := inventoryv1.NewAgentInventoryServiceClient(conn)

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiToken})
	perRPCCreds := grpc.PerRPCCredentials(oauth.TokenSource{TokenSource: tokenSource})
	if !tlsEnabled {
		perRPCCreds = grpc.PerRPCCredentials(grpcutils.InsecureDangerousTokenSource{TokenSource: tokenSource})
	}

	return Client{
		OrganizationId: organizationId,
		client:         c,
		logger:         logger,
		callOpts: []grpc.CallOption{
			perRPCCreds,
			grpc.WaitForReady(true),
			// CRD-heavy clusters produce multi-MB schema snapshots; compress and
			// lift the 4MB default send cap so the push doesn't ResourceExhausted.
			grpc.UseCompressor(gzip.Name),
			grpc.MaxCallSendMsgSize(math.MaxInt32),
		},
		callTimeout: defaultCallTimeout,
	}
}

// PutClusterResources replaces the CP's view of watchable GVKs for this
// agent's environment. Caller must pre-filter to GVKs the agent can watch.
func (c Client) PutClusterResources(ctx context.Context, resources []testkube.ClusterResource) error {
	if resources == nil {
		resources = []testkube.ClusterResource{}
	}
	payload, err := json.Marshal(resources)
	if err != nil {
		return fmt.Errorf("json encode cluster resources: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organization-id", c.OrganizationId)

	if _, err := c.client.PutClusterResources(callCtx, &inventoryv1.PutClusterResourcesRequest{
		Payload: payload,
	}, c.callOpts...); err != nil {
		return fmt.Errorf("push cluster resources: %w", err)
	}
	return nil
}
