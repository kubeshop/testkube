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

type Client interface {
	Get(ctx context.Context, id string) chan events.LogResponse
}

type Stream interface {
	StreamInitializer
	StreamPusher
	StreamTrigger
	StreamGetter
}

type StreamMetadata struct {
	Name string
}

type StreamInitializer interface {
	// Init creates or updates stream on demand
	Init(ctx context.Context) (meta StreamMetadata, err error)
}

type StreamPusher interface {
	// Push sends logs to log stream
	Push(ctx context.Context, chunk events.Log) error
	// PushBytes sends RAW bytes to log stream, developer is responsible for marshaling valid data
	PushBytes(ctx context.Context, chunk []byte) error
}

// LogStream is a single log stream chunk with possible errors
type StreamGetter interface {
	// Init creates or updates stream on demand
	Get(ctx context.Context) (chan events.LogResponse, error)
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
	Start(ctx context.Context) (StreamResponse, error)
	// Trigger stop event
	Stop(ctx context.Context) (StreamResponse, error)
}
