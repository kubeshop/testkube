package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/local"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/version"
)

const (
	connectionTimeout          = 10 * time.Second
	apiKeyMeta                 = "api-key"
	organizationIdMetadataName = "organization-id"
	environmentIdMetadataName  = "environment-id"
	agentIdMetadataName        = "agent-id"
	// The backoff values chosen here are copied from an example in the
	// gRPC documentation and represent a starting point that may be
	// iterated on as we learn more about the connection issues faced
	// by customers.
	// - https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md
	backoffDelay      = 1 * time.Second
	backoffMultiplier = 1.6
	backoffJitter     = 0.2
	backoffMaxDelay   = 120 * time.Second

	GRPCKeepaliveTime                = 10 * time.Second
	GRPCKeepaliveTimeout             = GRPCKeepaliveTime / 2
	GRPCKeepalivePermitWithoutStream = true
)

// NewGRPCConnection keeps backward compatibility, tracing disabled by default.
func NewGRPCConnection(
	ctx context.Context,
	isInsecure bool,
	skipVerify bool,
	server string,
	caFile string,
	logger *zap.SugaredLogger,
) (*grpc.ClientConn, error) {
	return NewGRPCConnectionWithTracing(ctx, isInsecure, skipVerify, server, caFile, logger, false)
}

// NewGRPCConnectionWithTracing creates a gRPC client and optionally instruments it with OpenTelemetry (non-deprecated stats handler).
func NewGRPCConnectionWithTracing(
	ctx context.Context,
	isInsecure bool,
	skipVerify bool,
	server string,
	caFile string,
	logger *zap.SugaredLogger,
	enableTracing bool,
) (*grpc.ClientConn, error) {
	// Build dial options
	opts := []grpc.DialOption{
		grpc.WithUserAgent(version.Version + "/" + version.Commit),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                GRPCKeepaliveTime,
			Timeout:             GRPCKeepaliveTimeout,
			PermitWithoutStream: GRPCKeepalivePermitWithoutStream,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  backoffDelay,
				Multiplier: backoffMultiplier,
				Jitter:     backoffJitter,
				MaxDelay:   backoffMaxDelay,
			},
			MinConnectTimeout: connectionTimeout,
		}),
		grpc.WithChainStreamInterceptor(
			grpczap.StreamClientInterceptor(logger.Desugar()),
		),
		grpc.WithChainUnaryInterceptor(
			grpczap.UnaryClientInterceptor(logger.Desugar()),
		),
	}
	// Conditionally add OpenTelemetry (non-deprecated) stats handler
	if enableTracing {
		opts = append(opts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	}

	// CONNECTION SECURITY
	// Here we're attempting to enforce some level of security here with the intention to
	// eventually just totally remove the ability to connect insecurely between Agents
	// and the Control Plane.
	// The logic for the below section is intended to be as follows:
	// 	1. Attempt to connect using TLS with either:
	// 		a. System Root CAs, or
	// 		b. Optionally provided CA from a PEM file.
	// 	2. Attempt to connect with a local only connection, for standalone Agents.
	// 	3. If secure connection fails then:
	// 		a. If skipVerify is set attempt to create a TLS connection without certificate verification (NOT RECOMMENDED!).
	// 		b. If skipVerify fails and insecure is set attempt to create a connection without TLS (EVEN MORE NOT RECOMMENDED!).
	//
	// All of step 2 should be removed in the future when Control Planes are required to create TLS connections.

	// Default credentials using the system CAs to verify server certificates.
	//certPool, err := x509.SystemCertPool()
	//if err != nil {
	//	return nil, err
	//}
	//creds := credentials.NewClientTLSFromCert(certPool, "")
	//
	//// If a CA certificate file is passed then use that CA to verify server certificates.
	//if caFile != "" {
	//	creds, err = credentials.NewClientTLSFromFile(caFile, "")
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	// Attempt to use a TLS connection.
	//tlsDialOptions := append(opts, grpc.WithTransportCredentials(creds))
	//client, err := attemptConnection(ctx, server, tlsDialOptions...)
	//// WARNING, checking for no error to early return with a secure client before attempting with local client.
	//if err == nil {
	//	logger.Info("Using TLS gRPC connection")
	//	return client, nil
	//}

	// Attempt to use a Local connection (for our usage only local TCP connections will work).
	localDialOptions := append(opts, grpc.WithTransportCredentials(local.NewCredentials()))
	client, err := attemptConnection(ctx, server, localDialOptions...)
	// WARNING, checking for no error to early return with a local client before descending into madness.
	if err == nil {
		logger.Info("Using local gRPC connection")
		return client, nil
	}

	// The following cases exist purely for backwards compatibility.
	// They should be removed once TLS is enforced on Control Plane gRPC servers.
	if skipVerify {
		skipVerifyDialOptions := append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: skipVerify,
		})))
		client, err = attemptConnection(ctx, server, skipVerifyDialOptions...)
		// WARNING, checking for no error to early return with an insecure (MitM is possible with skip verify) client before descending further into madness.
		if err == nil {
			logger.Error("Using TLS with no certificate verification for gRPC connection")
			return client, nil
		}
	}
	if isInsecure {
		insecureDialOptions := append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		client, err = attemptConnection(ctx, server, insecureDialOptions...)
		// WARNING, checking for no error to early return with an insecure client, this is madness.
		if err == nil {
			logger.Error("Using insecure gRPC connection")
			return client, nil
		}
	}

	return nil, err
}

func attemptConnection(ctx context.Context, url string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	client, err := grpc.NewClient(url, opts...)
	if err != nil {
		return client, fmt.Errorf("create new grpc client: %w", err)
	}
	// Wait for connection to go ready.
	for {
		s := client.GetState()
		if s == connectivity.Idle {
			client.Connect()
		}
		if s == connectivity.Ready {
			// Successfully connected.
			return client, nil
		}
		// Wait for transition away from current state.
		if !client.WaitForStateChange(ctx, s) {
			return nil, ctx.Err()
		}
	}
}

func AddAPIKeyMeta(ctx context.Context, apiKey string) context.Context {
	md := metadata.Pairs(apiKeyMeta, apiKey)
	return metadata.NewOutgoingContext(ctx, md)
}

func AddMetadata(ctx context.Context, apiKey, orgID, envID, agentID string) context.Context {
	md := metadata.Pairs(
		apiKeyMeta, apiKey,
		organizationIdMetadataName, orgID,
		environmentIdMetadataName, envID,
		agentIdMetadataName, agentID,
	)
	return metadata.NewOutgoingContext(ctx, md)
}
