package services

import (
	"github.com/kubeshop/testkube/pkg/event/bus"

	"go.uber.org/zap"
)

type Service interface {
	Bus() bus.Bus
	Logger() *zap.SugaredLogger
}

type service struct {
	bus    bus.Bus
	logger *zap.SugaredLogger
}

func (s *service) Bus() bus.Bus {
	return s.bus
}

func (s *service) Logger() *zap.SugaredLogger {
	return s.logger
}

func NewService(bus bus.Bus, logger *zap.SugaredLogger) Service {
	return &service{
		bus:    bus,
		logger: logger,
	}
}
