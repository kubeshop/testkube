package state

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go/jetstream"
)

// NewState creates new state storage
func NewState(kv jetstream.KeyValue) Interface {
	return &State{
		kv: kv,
	}
}

// State is a state storage based on NATS KV store
type State struct {
	kv jetstream.KeyValue
}

// Get returns state for given key - executionId
func (s State) Get(ctx context.Context, key string) (LogState, error) {
	state, err := s.kv.Get(ctx, key)
	if err != nil {
		return LogStateUnknown, err
	}

	if len(state.Value()) == 0 {
		return LogStateUnknown, errors.New("no state found")
	}

	return LogState(state.Value()[0]), nil

}

// Put puts state for given key - executionId
func (s State) Put(ctx context.Context, key string, state LogState) error {
	_, err := s.kv.Put(ctx, key, []byte{byte(state)})
	return err
}
