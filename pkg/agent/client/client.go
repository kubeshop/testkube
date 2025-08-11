package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/version"
)

const (
	initialConnectionTimeout = 10 * time.Second
	apiKeyMeta               = "api-key"

	GRPCKeepaliveTime                = 10 * time.Second
	GRPCKeepaliveTimeout             = GRPCKeepaliveTime / 2
	GRPCKeepalivePermitWithoutStream = true
)

func NewGRPCConnection(
	ctx context.Context,
	isInsecure bool,
	skipVerify bool,
	server string,
	certFile, keyFile, caFile string,
	logger *zap.SugaredLogger,
) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, initialConnectionTimeout)
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
	client, err := grpc.NewClient(server,
		grpc.WithUserAgent(userAgent),
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(kacp),
	)
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
