package resolvers

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/rand"
)

func CreateBusSubscription[T interface{}](
	ctx context.Context,
	r ResolverData,
	topic string,
	get func(r ResolverData) (T, error),
) (<-chan T, error) {
	ch := make(chan T, 1)

	// Load initial data
	initial, err := get(r)
	if err == nil {
		ch <- initial
	} else {
		r.Logger().Errorw("failed to get initial data for "+topic, err)
		return nil, err
	}

	// Setup queue
	queue := rand.String(30)
	err = r.Bus().SubscribeTopic(topic, queue, func(e testkube.Event) error {
		r.Logger().Debugf("graphql subscription event: %s %s %s", e.Type_, *e.Resource, e.ResourceId)
		result, err := get(r)
		if err != nil {
			r.Logger().Errorw("failed to get data after change for "+topic, err)
			return err
		}
		ch <- result
		return nil
	})

	if err == nil {
		r.Logger().Debug("graphql subscription: subscribed to " + topic)
		go func() {
			<-ctx.Done()
			_ = r.Bus().Unsubscribe(queue)
		}()
	} else {
		r.Logger().Errorw("graphql subscription: failed to subscribe to "+topic, err)
		return nil, err
	}

	return ch, nil
}
