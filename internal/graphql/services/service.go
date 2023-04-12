package services

import (
	"github.com/kubeshop/testkube/pkg/event/bus"
	"go.uber.org/zap"
)

type ServiceData interface {
	GetBus() bus.Bus
	GetLogger() *zap.SugaredLogger
}

type Service struct {
	Bus    bus.Bus
	Logger *zap.SugaredLogger
}

func (s *Service) GetBus() bus.Bus {
	return s.Bus
}

func (s *Service) GetLogger() *zap.SugaredLogger {
	return s.Logger
}
