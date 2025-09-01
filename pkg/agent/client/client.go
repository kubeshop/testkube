package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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
	certFile, keyFile, caFile string,
	logger *zap.SugaredLogger,
) (*grpc.ClientConn, error) {
	return NewGRPCConnectionWithTracing(ctx, isInsecure, skipVerify, server, certFile, keyFile, caFile, logger, false)
}

// NewGRPCConnectionWithTracing creates a gRPC client and optionally instruments it with OpenTelemetry (non-deprecated stats handler).
func NewGRPCConnectionWithTracing(
	ctx context.Context,
	isInsecure bool,
	skipVerify bool,
	server string,
	certFile, keyFile, caFile string,
	logger *zap.SugaredLogger,
	enableTracing bool,
) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if skipVerify {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		if certFile != "" && keyFile != "" {
			if err := clientCert(tlsConfig, certFile, keyFile); err != nil {
				return nil, err
			}
		}
		if caFile != "" {
			if err := rootCAs(tlsConfig, caFile); err != nil {
				return nil, err
			}
		}
	}

	creds := credentials.NewTLS(tlsConfig)
	if isInsecure {
		creds = insecure.NewCredentials()
	}

	kacp := keepalive.ClientParameters{
		Time:                GRPCKeepaliveTime,
		Timeout:             GRPCKeepaliveTimeout,
		PermitWithoutStream: GRPCKeepalivePermitWithoutStream,
	}

	userAgent := version.Version + "/" + version.Commit
	logger.Infow("initiating connection with control plane", "userAgent", userAgent, "server", server, "insecure", isInsecure, "skipVerify", skipVerify, "certFile", certFile, "keyFile", keyFile, "caFile", caFile)

	// Build dial options
	opts := []grpc.DialOption{
		grpc.WithUserAgent(userAgent),
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(kacp),
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

	client, err := grpc.NewClient(server, opts...)
	if err != nil {
		return client, fmt.Errorf("create new grpc client: %w", err)
	}
	var eg errgroup.Group
	eg.Go(func() error {
		if !client.WaitForStateChange(ctx, connectivity.Ready) {
			return context.DeadlineExceeded
		}
		return nil
	})
	client.Connect()
	if err := eg.Wait(); err != nil {
		return client, fmt.Errorf("connection did not go ready: %w", err)
	}
	return client, nil
}

func rootCAs(tlsConfig *tls.Config, file ...string) error {
	pool := x509.NewCertPool()
	for _, f := range file {
		rootPEM, err := os.ReadFile(f)
		if err != nil || rootPEM == nil {
			return fmt.Errorf("agent: error loading or parsing rootCA file: %v", err)
		}
		ok := pool.AppendCertsFromPEM(rootPEM)
		if !ok {
			return fmt.Errorf("agent: failed to parse root certificate from %q", f)
		}
	}
	tlsConfig.RootCAs = pool
	return nil
}

func clientCert(tlsConfig *tls.Config, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("agent: error loading client certificate: %v", err)
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("agent: error parsing client certificate: %v", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}
	return nil
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
