package main

import (
	"context"
	"errors"

	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/oklog/run"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs"
	"github.com/kubeshop/testkube/pkg/logs/adapter"
	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/config"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/pb"
	"github.com/kubeshop/testkube/pkg/logs/repository"
	"github.com/kubeshop/testkube/pkg/logs/state"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

func newStorageClient(cfg *config.Config) *minio.Client {
	opts := minio.GetTLSOptions(cfg.StorageSSL, cfg.StorageSkipVerify, cfg.StorageCertFile, cfg.StorageKeyFile, cfg.StorageCAFile)
	return minio.NewClient(
		cfg.StorageEndpoint,
		cfg.StorageAccessKeyID,
		cfg.StorageSecretAccessKey,
		cfg.StorageRegion,
		cfg.StorageToken,
		cfg.StorageBucket,
		opts...,
	)
}

func main() {
	var g run.Group

	ctx, cancel := context.WithCancel(context.Background())

	cfg := Must(config.Get())

	mode := common.ModeStandalone
	if cfg.TestkubeProAPIKey != "" {
		mode = common.ModeAgent
	}

	log := log.DefaultLogger.With("service", "logs-service-init", "mode", mode)

	log.Debugw("starting logs service", "config", cfg)

	// Event bus
	nc := Must(bus.NewNATSConnection(bus.ConnectionConfig{
		NatsURI:            cfg.NatsURI,
		NatsSecure:         cfg.NatsSecure,
		NatsSkipVerify:     cfg.NatsSkipVerify,
		NatsCertFile:       cfg.NatsCertFile,
		NatsKeyFile:        cfg.NatsKeyFile,
		NatsCAFile:         cfg.NatsCAFile,
		NatsConnectTimeout: cfg.NatsConnectTimeout,
	}))
	defer func() {
		log.Infof("closing nats connection")
		nc.Close()
	}()

	js := Must(jetstream.New(nc))
	logStream := Must(client.NewNatsLogStream(nc))

	minioClient := newStorageClient(cfg)
	if err := minioClient.Connect(); err != nil {
		log.Fatalw("error connecting to minio", "error", err)
	}

	if err := minioClient.SetExpirationPolicy(cfg.StorageExpiration); err != nil {
		log.Warnw("error setting expiration policy", "error", err)
	}

	kv := Must(js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: cfg.KVBucketName}))
	state := state.NewState(kv)

	svc := logs.NewLogsService(nc, js, state, logStream).
		WithHttpAddress(cfg.HttpAddress).
		WithGrpcAddress(cfg.GrpcAddress).
		WithLogsRepositoryFactory(repository.NewJsMinioFactory(minioClient, cfg.StorageBucket, logStream))

	if cfg.Debug {
		svc.AddAdapter(adapter.NewDebugAdapter())
	}

	creds, err := newGRPCTransportCredentials(cfg)
	if err != nil {
		log.Fatalw("error getting tls credentials", "error", err)
	}

	// add given log adapter depends from mode
	switch mode {

	case common.ModeAgent:
		grpcConn, err := agent.NewGRPCConnection(ctx, cfg.TestkubeProTLSInsecure, cfg.TestkubeProSkipVerify, cfg.TestkubeProURL+cfg.TestkubeProLogsPath, log)
		ui.ExitOnError("error creating gRPC connection for logs service", err)
		log.Infow("grpc connection state", "state", grpcConn.GetState())

		defer grpcConn.Close()
		grpcClient := pb.NewCloudLogsServiceClient(grpcConn)

		// write metadata to the stream context
		md := metadata.Pairs("api-key", cfg.TestkubeProAPIKey, "execution-id", "test-id")
		ctx = metadata.NewOutgoingContext(ctx, md)

		stream, err := grpcClient.Stream(ctx)
		ui.ExitOnError("error creating gRPC logs stream", err)

		err = stream.Send(pb.MapToPB(*events.NewLog("aaaaa")))
		ui.ExitOnError("error sending test message", err)

		err = stream.CloseSend()
		ui.ExitOnError("error sending test message", err)

		log.Debugw("test message sent")

		cloudAdapter := adapter.NewCloudAdapter(grpcClient, cfg.TestkubeProAPIKey)
		svc.AddAdapter(cloudAdapter)
		log.Debugw("cloud adapter configured", "url", cfg.TestkubeProURL+cfg.TestkubeProLogsPath)

	case common.ModeStandalone:
		minioAdapter, err := adapter.NewMinioAdapter(cfg.StorageEndpoint,
			cfg.StorageAccessKeyID,
			cfg.StorageSecretAccessKey,
			cfg.StorageRegion,
			cfg.StorageToken,
			cfg.StorageBucket,
			cfg.StorageSSL,
			cfg.StorageSkipVerify,
			cfg.StorageCertFile,
			cfg.StorageKeyFile,
			cfg.StorageCAFile)

		if err != nil {
			log.Errorw("error creating minio adapter", "error", err)
		}
		svc.AddAdapter(minioAdapter)
		log.Infow("minio adapter configured", "bucket", cfg.StorageBucket, "endpoint", cfg.StorageEndpoint)
	}

	g.Add(func() error {
		err := interrupt(log, ctx)
		return err
	}, func(error) {
		log.Warnf("interrupt signal received, canceling context")
		cancel()
	})

	g.Add(func() error {
		return svc.Run(ctx)
	}, func(error) {
		err := svc.Shutdown(ctx)
		if err != nil {
			log.Errorw("error shutting down logs service", "error", err)
		}
		log.Warn("logs service shutdown")
	})

	g.Add(func() error {
		return svc.RunGRPCServer(ctx, creds)
	}, func(error) {
		cancel()
	})

	// We need to do a http health check to be backward compatible with Kubernetes below 1.25
	g.Add(func() error {
		return svc.RunHealthCheckHandler(ctx)
	}, func(error) {
		cancel()
	})

	if err := g.Run(); err != nil {
		log.Warnw("logs service run group returned an error", "error", err)
	}

	log.Infof("exiting")
}

func interrupt(logger *zap.SugaredLogger, ctx context.Context) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-c:
		return errors.New("signal received" + s.String())
	case <-ctx.Done():
		return context.Canceled
	}
}

// Must helper function to panic on error
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func newGRPCTransportCredentials(cfg *config.Config) (credentials.TransportCredentials, error) {
	return logs.GetGrpcTransportCredentials(logs.GrpcConnectionConfig{
		Secure:       cfg.GrpcSecure,
		ClientAuth:   cfg.GrpcClientAuth,
		CertFile:     cfg.GrpcCertFile,
		KeyFile:      cfg.GrpcKeyFile,
		ClientCAFile: cfg.GrpcClientCAFile,
	})
}
