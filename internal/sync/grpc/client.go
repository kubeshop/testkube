// Package grpc provides simple to use functions that call Contrl Plane gRPC endpoints
// for updating the Control Plane about changes made to synchronised objects.
package grpc

import (
	"context"
	"time"

	"github.com/cloudflare/backoff"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

const defaultCallTimeout = time.Second * 30

type Client struct {
	OrganisationId string

	client      syncv1.SyncServiceClient
	logger      *zap.SugaredLogger
	callOpts    []grpc.CallOption
	callTimeout time.Duration
}

func NewClient(conn grpc.ClientConnInterface, logger *zap.SugaredLogger, apiToken, organisationId string) Client {
	c := syncv1.NewSyncServiceClient(conn)
	return Client{
		OrganisationId: organisationId,

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

// IsSupported attempts to contact the Control Plane to determine whether or not there is an implementation
// of the required server to support this client.
// It will block until it receives either a successful response, returning true and indicating that the server
// supports this client.
// Or it receives an "Unimplemented" response, returning false and indicating that the server does not support
// this client and a fallback should be used instead.
// In the event of any other error, such as an authentication failure or a network failure, it will continue
// to loop, using a backoff mechanism, until one of the above cases is satisfied.
func (c Client) IsSupported(ctx context.Context) bool {
	b := backoff.New(backoff.DefaultMaxDuration, backoff.DefaultInterval)
	for {
		if ctx.Err() != nil {
			return false
		}
		// Execute with our own call timeout context to prevent stalling out.
		callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
		// Add metadata to the call.
		callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)
		// Attempt to call delete with a workflow that should not exist.
		// This means that we should expect to receive a NotFound response
		// from the server, and any other response represents an error of
		// some form.
		testId := "id-that-should-not-exist-vgyxwqavpd"
		_, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
			Id: &syncv1.DeleteRequest_TestWorkflow{
				TestWorkflow: &syncv1.TestWorkflowId{
					Id: &testId,
				},
			},
		}, c.callOpts...)
		cancel()
		code, ok := status.FromError(err)
		switch {
		case ok && code.Code() == codes.Unimplemented:
			// Server does not have the implementation for this client.
			return false
		case ok && code.Code() == codes.NotFound:
			// Correctly implemented server.
			return true
		case err != nil:
			c.logger.Warnw("Failed to check if server supports polling execution updates, backing off before retrying.",
				"backoff", b.Duration(),
				"error", err)
			// In the event of an error wait for backoff before trying again.
			<-time.After(b.Duration())
			continue
		}

		// Server has implementation but for some reason it accepted our Id that should not exist...
		c.logger.Warnw("Server has support but claims to have deleted an object that should not exist.",
			"object type", "TestWorkflow",
			"object id", testId)
		return true
	}
}
