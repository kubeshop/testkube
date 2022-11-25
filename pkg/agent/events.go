package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var _ common.ListenerLoader = &Agent{}

func (ag *Agent) Kind() string {
	return "agent"
}

func (ag *Agent) Load() (listeners common.Listeners, err error) {
	listeners = append(listeners, ag)

	return listeners, nil
}

func (ag *Agent) Name() string {
	return "agent"
}

func (ag *Agent) Selector() string {
	return ""
}

func (ag *Agent) Events() []testkube.EventType {
	return testkube.AllEventTypes
}
func (ag *Agent) Metadata() map[string]string {
	return map[string]string{
		"name":     ag.Name(),
		"selector": "",
		"events":   fmt.Sprintf("%v", ag.Events()),
	}
}

func (ag *Agent) Notify(event testkube.Event) (result testkube.EventResult) {
	// Non blocking send
	select {
	case ag.events <- event:
		return testkube.NewSuccessEventResult(event.Id, "message sent to websocket clients")
	default:
		return testkube.NewFailedEventResult(event.Id, errors.New("message not sent"))
	}
}

func (ag *Agent) RunEventLoop(ctx context.Context) error {
	var opts []grpc.CallOption
	md := metadata.Pairs(apiKey, ag.apiKey)
	ctx = metadata.NewOutgoingContext(ctx, md)

	//TODO figure out how to retry this method in case of network failure
	// creates a new Stream from the client side. ctx is used for the lifetime of the stream.
	stream, err := ag.client.Send(ctx, opts...)
	if err != nil {
		ag.logger.Errorf("failed to execute: %w", err)
		return fmt.Errorf("failed to setup stream: %w", err)
	}
	for {
		ev := <-ag.events
		b, err := json.Marshal(ev)
		if err != nil {
			continue
		}

		msg := &cloud.WebsocketData{Opcode: cloud.Opcode_TEXT_FRAME, Body: b}
		err = stream.Send(msg)
		if err != nil {
			ag.logger.Errorf("websocket stream send: %w", err)
			return err
		}
	}
}
