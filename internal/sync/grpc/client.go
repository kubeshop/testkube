// Package grpc provides simple to use functions that call Contrl Plane gRPC endpoints
// for updating the Control Plane about changes made to synchronised objects.
package grpc

import (
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"

	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

const defaultCallTimeout = time.Second * 30

type Client struct {
	OrganizationId string

	client      syncv1.SyncServiceClient
	logger      *zap.SugaredLogger
	callOpts    []grpc.CallOption
	callTimeout time.Duration
}

func NewClient(conn grpc.ClientConnInterface, logger *zap.SugaredLogger, apiToken, organizationId string) Client {
	c := syncv1.NewSyncServiceClient(conn)
	return Client{
		OrganizationId: organizationId,

		client: c,
		logger: logger,
		callOpts: []grpc.CallOption{
			// Note: This requires TLS to be correctly configured, otherwise the gRPC library will
			// abort the connection. It is not secure to send authentication tokens over an
			// unencrypted connection so this is appropriate behaviour.
			grpc.PerRPCCredentials(oauth.TokenSource{
				TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: apiToken,
				}),
			}),
			// In the event of a transient failure on the server wait for it to come back rather than
			// failing immediately.
			grpc.WaitForReady(true),
		},
		callTimeout: defaultCallTimeout,
	}
}
