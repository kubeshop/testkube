package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/pb"
)

const (
	buffer          = 100
	requestDeadline = time.Minute * 5
)

// NewGrpcClient imlpements getter interface for log stream for given ID
func NewGrpcClient(address string, creds credentials.TransportCredentials) StreamGetter {
	return &GrpcClient{
		log:     log.DefaultLogger.With("service", "logs-grpc-client"),
		creds:   creds,
		address: address,
	}
}

type GrpcClient struct {
	log     *zap.SugaredLogger
	creds   credentials.TransportCredentials
	address string
}

// Get returns channel with log stream chunks for given execution id connects through GRPC to log service
func (c GrpcClient) Get(ctx context.Context, id string) (chan events.LogResponse, error) {
	ch := make(chan events.LogResponse, buffer)

	log := c.log.With("id", id)

	log.Debugw("getting logs", "address", c.address)

	go func() {
		// Contact the server and print out its response.
		ctx, cancel := context.WithTimeout(context.Background(), requestDeadline)
		defer cancel()
		defer close(ch)

		// TODO add TLS to GRPC client
		creds := insecure.NewCredentials()
		if c.creds != nil {
			creds = c.creds
		}

		conn, err := grpc.Dial(c.address, grpc.WithTransportCredentials(creds))
		if err != nil {
			ch <- events.LogResponse{Error: err}
			return
		}
		defer conn.Close()
		log.Debugw("connected to grpc server")

		client := pb.NewLogsServiceClient(conn)

		r, err := client.Logs(ctx, &pb.LogRequest{ExecutionId: id})
		if err != nil {
			ch <- events.LogResponse{Error: err}
			log.Errorw("error getting logs", "error", err)
			return
		}

		log.Debugw("client start streaming")
		defer func() {
			log.Debugw("client stopped streaming")
		}()

		for {
			l, err := r.Recv()
			if err == io.EOF {
				log.Infow("client stream finished", "error", err)
				return
			} else if err != nil {
				ch <- events.LogResponse{Error: err}
				log.Errorw("error receiving log response", "error", err)
				return
			}

			logChunk := pb.MapFromPB(l)

			// catch finish event
			if events.IsFinished(&logChunk) {
				log.Infow("received finish", "log", l)
				return
			}

			log.Debugw("grpc client log", "log", l)
			// send to the channel
			ch <- events.LogResponse{Log: logChunk}
		}
	}()

	return ch, nil
}

// GrpcConnectionConfig contains GRPC connection parameters
type GrpcConnectionConfig struct {
	Secure     bool
	SkipVerify bool
	CertFile   string
	KeyFile    string
	CAFile     string
}

// GetGrpcTransportCredentials returns transport credentials for GRPC connection config
func GetGrpcTransportCredentials(cfg GrpcConnectionConfig) (credentials.TransportCredentials, error) {
	var creds credentials.TransportCredentials

	if cfg.Secure {
		var tlsConfig tls.Config

		if cfg.SkipVerify {
			tlsConfig.InsecureSkipVerify = true
		} else {
			if cfg.CertFile != "" && cfg.KeyFile != "" {
				cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
				if err != nil {
					return nil, err
				}

				tlsConfig.Certificates = []tls.Certificate{cert}
			}

			if cfg.CAFile != "" {
				caCertificate, err := os.ReadFile(cfg.CAFile)
				if err != nil {
					return nil, err
				}

				certPool := x509.NewCertPool()
				if !certPool.AppendCertsFromPEM(caCertificate) {
					return nil, fmt.Errorf("failed to add server CA's certificate")
				}

				tlsConfig.RootCAs = certPool
			}
		}

		creds = credentials.NewTLS(&tlsConfig)
	}

	return creds, nil
}
