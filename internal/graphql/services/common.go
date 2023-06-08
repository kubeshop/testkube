package services

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/rand"
)

func Map[T interface{}, U interface{}](list []T, mapper func(item T) U) []U {
	result := make([]U, len(list))
	for i, item := range list {
		result[i] = mapper(item)
	}
	return result
}

func HandleSubscription[T Service, U interface{}](
	ctx context.Context,
	topic string,
	s T,
	get func() (U, error),
) (<-chan U, error) {
	ch := make(chan U, 1)

	// Load initial data
	initial, err := get()
	if err == nil {
		ch <- initial
	} else {
		s.Logger().Errorw("failed to get initial data for "+topic, err)
		return nil, err
	}

	// Setup queue
	queue := rand.String(30)
	err = s.Bus().SubscribeTopic(topic, queue, func(e testkube.Event) error {
		s.Logger().Debugf("graphql subscription event: %s %s %s", e.Type_, *e.Resource, e.ResourceId)
		result, err := get()
		if err != nil {
			s.Logger().Errorw("failed to get data after change for "+topic, err)
			return err
		}
		ch <- result
		return nil
	})

	if err == nil {
		s.Logger().Debug("graphql subscription: subscribed to " + topic)
		go func() {
			<-ctx.Done()
			_ = s.Bus().Unsubscribe(queue)
			close(ch)
		}()
	} else {
		s.Logger().Errorw("graphql subscription: failed to subscribe to "+topic, err)
		return nil, err
	}

	return ch, nil
}
