package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
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

	"github.com/kubeshop/testkube/pkg/grpcutils"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	connectionTimeout          = 3 * time.Second
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
	// Last-resort recycle of long-lived agent streams, for targets where the client
	// cannot see the individual replicas (an Ingress, or a plain ClusticIP Service).
	// Recycling breaks live notification streams, which then resume from the agent's
	// seqNo replay buffer, so keep it rare: prefer resolving replicas via
	// GRPCLoadBalancingPolicy below, which spreads streams without breaking any.
	GRPCMaxConnectionAge      = 30 * time.Minute
	GRPCMaxConnectionAgeGrace = 30 * time.Second
	// How long to stay in TransientFailure before giving up on this dial
	// attempt so the next credential mode (or retry) can run.
	transientFailureTimeout = 2 * time.Second

	// round_robin opens a subchannel per resolved address and assigns each new
	// stream to the next one, so an agent's long-lived streams spread over every
	// replica a `dns:///` target resolves to and a dead replica only takes down its
	// own streams. Against a single-address target (Ingress, ClusterIP VIP) it
	// degrades to the pick_first behaviour it replaces.
	GRPCLoadBalancingPolicy = `{"loadBalancingConfig":[{"round_robin":{}}]}`

	// grpc-go's DNS resolver does not poll: after a successful lookup it blocks
	// until something calls ResolveNow, which the LB policy only does when a
	// subchannel fails. So a scale-up is invisible — the new Pod is never dialled
	// and receives no streams. Re-resolving on this interval adds a subchannel for
	// each new replica without touching any established stream. The resolver
	// rate-limits re-resolution to 30s (dns.MinResolutionInterval), so this must
	// stay above that to have any effect.
	GRPCResolveInterval = 60 * time.Second
)

// Build dial options
var dialOpts = []grpc.DialOption{
	grpc.WithUserAgent(version.Version + "/" + version.Commit),
	grpc.WithDefaultServiceConfig(GRPCLoadBalancingPolicy),
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
}

func init() {
	if r := refreshingDNSResolver(GRPCResolveInterval); r != nil {
		dialOpts = append(dialOpts, grpc.WithResolvers(r))
	}
}

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
	opts := append(dialOpts,
		grpc.WithChainStreamInterceptor(
			logging.StreamClientInterceptor(grpcutils.ZapGRPCLogger(logger.Desugar())),
		),
		grpc.WithChainUnaryInterceptor(
			logging.UnaryClientInterceptor(grpcutils.ZapGRPCLogger(logger.Desugar())),
		),
	)

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
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	creds := credentials.NewClientTLSFromCert(certPool, "")

	// If a CA certificate file is passed then use that CA to verify server certificates.
	// A leaf/self-signed PEM without CA:TRUE fails AppendCertsFromPEM; keep going so
	// skipVerify / insecure fallbacks can still connect (typical for Ingress lab certs).
	if caFile != "" {
		caCreds, caErr := credentials.NewClientTLSFromFile(caFile, "")
		if caErr != nil {
			logger.Warnw("failed to load TESTKUBE_PRO_CA_FILE, continuing with TLS fallbacks", "error", caErr)
		} else {
			creds = caCreds
		}
	}

	// If the caller explicitly requested plaintext, try that first. TLS-first
	// against an HTTP Ingress (:80) or a self-signed :443 sits in Connecting
	// until MinConnectTimeout and used to burn the whole startup budget.
	if isInsecure {
		insecureDialOptions := append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		// nginx grpc_pass often stays Connecting until the first RPC.
		client, err := attemptConnection(ctx, server, true, insecureDialOptions...)
		if err == nil {
			logger.Error("Using insecure gRPC connection")
			return client, nil
		}
	}

	// Attempt to use a TLS connection.
	tlsDialOptions := append(opts, grpc.WithTransportCredentials(creds))
	// If skipVerify is set, do not accept Connecting here: grpc-go defers the
	// TLS handshake until the first RPC, so a self-signed Ingress would look
	// successful and skipVerify would never run. With a real CA, Connecting is
	// fine — the first RPC completes the handshake.
	client, err := attemptConnection(ctx, server, !skipVerify, tlsDialOptions...)
	if err == nil {
		logger.Info("Using TLS gRPC connection")
		return client, nil
	}

	if isLocalTarget(server) {
		localDialOptions := append(opts, grpc.WithTransportCredentials(local.NewCredentials()))
		client, err = attemptConnection(ctx, server, true, localDialOptions...)
		if err == nil {
			logger.Info("Using local gRPC connection")
			return client, nil
		}
	}

	if skipVerify {
		skipVerifyDialOptions := append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: skipVerify,
		})))
		client, err = attemptConnection(ctx, server, true, skipVerifyDialOptions...)
		if err == nil {
			logger.Error("Using TLS with no certificate verification for gRPC connection")
			return client, nil
		}
	}

	return nil, err
}

func isLocalTarget(server string) bool {
	host := server
	if h, _, err := net.SplitHostPort(server); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

func attemptConnection(ctx context.Context, url string, acceptUnready bool, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	client, err := grpc.NewClient(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("create new grpc client: %w", err)
	}
	closeFailed := func(err error) (*grpc.ClientConn, error) {
		_ = client.Close()
		return nil, err
	}
	transientDeadline := time.Now().Add(transientFailureTimeout)
	for {
		s := client.GetState()
		if s == connectivity.Idle {
			client.Connect()
		}
		if s == connectivity.Ready {
			return client, nil
		}
		waitCtx := ctx
		var waitCancel context.CancelFunc
		if s == connectivity.TransientFailure {
			waitCtx, waitCancel = context.WithDeadline(ctx, transientDeadline)
		}
		changed := client.WaitForStateChange(waitCtx, s)
		if waitCancel != nil {
			waitCancel()
		}
		if changed {
			continue
		}
		if s == connectivity.TransientFailure {
			return closeFailed(fmt.Errorf("grpc connection stuck in TransientFailure"))
		}
		// nginx grpc_pass often leaves skipVerify/plaintext in Connecting until
		// the first RPC. Verified TLS must not take that shortcut or skipVerify
		// never runs against a self-signed Ingress.
		if acceptUnready {
			return client, nil
		}
		return closeFailed(fmt.Errorf("wait for grpc ready: context deadline exceeded"))
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

// NewVeryInsecureGRPCClientDoNotUseThisClientUnlessYouAreReallySureYouKnowWhatYouAreDoing
// creates an insecure gRPC connection to a server. By default it will create a TLS connection
// that will perform no certificate verification, whilst this probably looks kind of like
// a TLS connection it actually provides no security whatsoever.
// Optionally you can choose to make your gRPC connection even less secure by setting the
// isInsecure flag to true. In this mode the gRPC connection won't even have an unverified
// TLS configuration and all communication will be in the clear.
// DO NOT USE THIS FUNCTION!
func NewVeryInsecureGRPCClientDoNotUseThisClientUnlessYouAreReallySureYouKnowWhatYouAreDoing(
	ctx context.Context,
	isInsecure bool,
	server string,
	logger *zap.SugaredLogger,
) (*grpc.ClientConn, error) {
	// Build dial options
	opts := append(dialOpts,
		grpc.WithChainStreamInterceptor(
			logging.StreamClientInterceptor(grpcutils.ZapGRPCLogger(logger.Desugar())),
		),
		grpc.WithChainUnaryInterceptor(
			logging.UnaryClientInterceptor(grpcutils.ZapGRPCLogger(logger.Desugar())),
		),
	)

	creds := credentials.NewTLS(&tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	})
	if isInsecure {
		creds = insecure.NewCredentials()
	}

	insecureDialOptions := append(opts, grpc.WithTransportCredentials(creds))

	return attemptConnection(ctx, server, true, insecureDialOptions...)
}
