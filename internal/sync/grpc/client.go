// Package grpc provides simple to use functions that call Contrl Plane gRPC endpoints
// for updating the Control Plane about changes made to synchronised objects.
package grpc

import (
	"context"
	"maps"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"

	syncagent "github.com/kubeshop/testkube/internal/sync"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/grpcutils"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

const defaultCallTimeout = time.Second * 30

type Client struct {
	OrganizationId string
	// EnvironmentId is the Environment of the agent token, where a resource syncs by default.
	EnvironmentId string

	client      syncv1.SyncServiceClient
	logger      *zap.SugaredLogger
	callOpts    []grpc.CallOption
	callTimeout time.Duration
}

func NewClient(conn grpc.ClientConnInterface, logger *zap.SugaredLogger, apiToken, organizationId, environmentId string, tlsEnabled bool) Client {
	c := syncv1.NewSyncServiceClient(conn)

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: apiToken,
	})
	perRPCCreds := grpc.PerRPCCredentials(oauth.TokenSource{
		TokenSource: tokenSource,
	})
	if !tlsEnabled {
		perRPCCreds = grpc.PerRPCCredentials(grpcutils.InsecureDangerousTokenSource{
			TokenSource: tokenSource,
		})
	}

	return Client{
		OrganizationId: organizationId,
		EnvironmentId:  environmentId,

		client: c,
		logger: logger,
		callOpts: []grpc.CallOption{
			// Prefer TLS-enforced credentials; when TLS is not configured fall back to an insecure token source.
			perRPCCreds,
			// In the event of a transient failure on the server wait for it to come back rather than
			// failing immediately.
			grpc.WaitForReady(true),
		},
		callTimeout: defaultCallTimeout,
	}
}

// callContext returns the context of one call to the Control Plane: our own call timeout, to
// prevent stalling out, and the metadata of the target.
//
// The environment-id header goes out only for an Environment other than the one of the agent
// token. The Control Plane already syncs into the token Environment without it, and a Control
// Plane without the sync-multi-namespace capability refuses the header.
func (c Client) callContext(ctx context.Context, target syncagent.Target) (context.Context, context.CancelFunc) {
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organization-id", c.OrganizationId)
	if target.EnvironmentID != "" && target.EnvironmentID != c.EnvironmentId {
		callCtx = metadata.AppendToOutgoingContext(callCtx, controlplaneclient.EnvironmentIdMetadataName, target.EnvironmentID)
	}
	return callCtx, cancel
}

// withSourceNamespace returns a copy of labels with syncagent.SourceNamespaceLabel set to
// namespace. It copies the map because the labels belong to the Kubernetes object in the cache of
// the controller, which the sync must not change. An empty namespace returns labels unchanged.
func withSourceNamespace(labels map[string]string, namespace string) map[string]string {
	if namespace == "" {
		return labels
	}
	result := maps.Clone(labels)
	if result == nil {
		result = make(map[string]string, 1)
	}
	result[syncagent.SourceNamespaceLabel] = namespace
	return result
}

// optionalNamespace returns nil for an empty namespace, so that the request leaves the field unset.
func optionalNamespace(namespace string) *string {
	if namespace == "" {
		return nil
	}
	return &namespace
}
