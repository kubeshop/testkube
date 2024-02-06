package adapter

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/pb"
)

var _ Adapter = &CloudAdapter{}

// NewCloudConsumer creates new CloudSubscriber which will send data to local MinIO bucket
func NewCloudAdapter(grpcConn pb.CloudLogsServiceClient, agentApiKey string) *CloudAdapter {
	return &CloudAdapter{
		client:      grpcConn,
		agentApiKey: agentApiKey,
		logger:      log.DefaultLogger.With("service", "logs-cloud-adapter"),
	}
}

type CloudAdapter struct {
	client      pb.CloudLogsServiceClient
	streams     sync.Map
	agentApiKey string
	logger      *zap.SugaredLogger
}

func (s *CloudAdapter) Init(ctx context.Context, id string) error {

	// write metadata to the stream context
	md := metadata.Pairs("api-key", s.agentApiKey, "execution-id", id)
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := s.client.Stream(ctx)
	if err != nil {
		return errors.Wrap(err, "can't init stream")
	}

	s.streams.Store(id, stream)

	return nil
}

func (s *CloudAdapter) Notify(ctx context.Context, id string, e events.Log) error {
	c, err := s.getStreamClient(id)
	if err != nil {
		return errors.Wrap(err, "can't get stream client for id: "+id)
	}

	return c.Send(pb.MapToPB(e))
}

func (s *CloudAdapter) Stop(ctx context.Context, id string) error {
	c, err := s.getStreamClient(id)
	if err != nil {
		return errors.Wrap(err, "can't get stream client for id: "+id)
	}

	resp, err := c.CloseAndRecv()
	if err != nil {
		return errors.Wrap(err, "closing log stream error")
	}
	s.logger.Debugw("closing response", "resp", resp, "id", id)

	s.streams.Delete(id)
	return nil
}

func (s *CloudAdapter) Name() string {
	return "cloud"
}

func (s *CloudAdapter) getStreamClient(id string) (client pb.CloudLogsService_StreamClient, err error) {
	c, ok := s.streams.Load(id)
	if !ok {
		return nil, errors.New("can't find initialized stream")
	}

	return c.(pb.CloudLogsService_StreamClient), nil
}
