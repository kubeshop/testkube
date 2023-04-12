package services

import (
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"
)

type MockService interface {
	Service
	BusMock() *bus.EventBusMock
	Reset()
}

type mockService struct {
	*service
}

func NewMockService() MockService {
	return &mockService{
		service: &service{
			bus:    bus.NewEventBusMock(),
			logger: log.DefaultLogger,
		},
	}
}

func (s *mockService) Reset() {
	_ = s.bus.Close()
	s.bus = bus.NewEventBusMock()
}

func (s *mockService) BusMock() *bus.EventBusMock {
	return s.bus.(*bus.EventBusMock)
}
