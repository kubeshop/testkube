package client

import (
	"context"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

const (
	StreamPrefix = "log"

	StartSubject = "events.logs.start"
	StopSubject  = "events.logs.stop"
)

//go:generate mockgen -destination=./mock_stream.go -package=client "github.com/kubeshop/testkube/pkg/logs/client" Stream
type Stream interface {
	StreamInitializer
	StreamPusher
	StreamTrigger
	StreamGetter
	StreamFinisher
	StreamNamer
}

//go:generate mockgen -destination=./mock_initializedstreampusher.go -package=client "github.com/kubeshop/testkube/pkg/logs/client" InitializedStreamPusher
type InitializedStreamPusher interface {
	StreamInitializer
	StreamPusher
}

//go:generate mockgen -destination=./mock_initializedstreamgetter.go -package=client "github.com/kubeshop/testkube/pkg/logs/client" InitializedStreamGetter
type InitializedStreamGetter interface {
	StreamInitializer
	StreamGetter
}

type StreamMetadata struct {
	Name string
}

type StreamInitializer interface {
	// Init creates or updates stream on demand
	Init(ctx context.Context, id string) (meta StreamMetadata, err error)
}

type StreamPusher interface {
	// Push sends logs to log stream
	Push(ctx context.Context, id string, log *events.Log) error
	// PushBytes sends RAW bytes to log stream, developer is responsible for marshaling valid data
	PushBytes(ctx context.Context, id string, bytes []byte) error
}

type StreamFinisher interface {
	// Finish sends termination log message
	Finish(ctx context.Context, id string) error
}

//go:generate mockgen -destination=./mock_namer.go -package=client "github.com/kubeshop/testkube/pkg/logs/client" StreamNamer
type StreamNamer interface {
	// Name returns stream name based on possible name groups
	Name(parts ...string) string
}

// StreamGetter interface for getting logs stream channel
//
//go:generate mockgen -destination=./mock_streamgetter.go -package=client "github.com/kubeshop/testkube/pkg/logs/client" StreamGetter
type StreamGetter interface {
	// Init creates or updates stream on demand
	Get(ctx context.Context, id string) (chan events.LogResponse, error)
}

type StreamConfigurer interface {
	// Init creates or updates stream on demand
	WithAddress(address string) Stream
}

type LogResponse struct {
	Log   events.Log
	Error error
}

type StreamResponse struct {
	Message []byte
	Error   bool
}

type StreamTrigger interface {
	// Trigger start event
	Start(ctx context.Context, id string) (StreamResponse, error)
	// Trigger stop event
	Stop(ctx context.Context, id string) (StreamResponse, error)
}
