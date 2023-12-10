package client

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/pb"
)

const (
	buffer = 100
)

func NewGrpcClient(address string) Client {
	return &GrpcClient{
		log:     log.DefaultLogger.With("service", "logs-grpc-client"),
		address: address,
	}
}

type GrpcClient struct {
	log     *zap.SugaredLogger
	address string
}

// Get returns channel with log stream chunks for given execution id connects through GRPC to log service
func (c GrpcClient) Get(ctx context.Context, id string) chan events.LogResponse {
	ch := make(chan events.LogResponse, buffer)
	log := c.log.With("id", id)

	log.Debugw("getting logs", "address", c.address)
	go func() {
		// Contact the server and print out its response.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		defer close(ch)

		// TODO add TLS to GRPC client
		conn, err := grpc.Dial(c.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		for {
			l, err := r.Recv()
			log.Debugw("received log chunk from client", "log", l, "error", err)
			if err == io.EOF {
				log.Debugw("client stream finished", "error", err)
				break
			} else if err != nil {
				log.Errorw("error receiving log response", "error", err)
				ch <- events.LogResponse{Error: err}
				continue
			}

			// send to the channel
			ch <- events.LogResponse{Log: pb.MapFromPB(l)}
		}
	}()

	return ch
}
