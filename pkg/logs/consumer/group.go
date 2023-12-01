package consumer

import "github.com/kubeshop/testkube/pkg/logs/events"

type ConsumerGroup struct {
	subscribers []Adapter
}

func (s *ConsumerGroup) Add(sub Adapter) {
	s.subscribers = append(s.subscribers, sub)
}

func (s *ConsumerGroup) NotifyAll(id string, event events.LogChunk) error {
	for _, sub := range s.subscribers {
		if err := sub.Notify(id, event); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsumerGroup) StopAll(id string) error {
	for _, sub := range s.subscribers {
		if err := sub.Stop(id); err != nil {
			return err
		}
	}
	return nil
}
