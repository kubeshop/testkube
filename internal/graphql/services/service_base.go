package services

import (
	"github.com/kubeshop/testkube/pkg/event/bus"

	"go.uber.org/zap"
)

type ServiceBase struct {
	Service
}

func (s *ServiceBase) Bus() bus.Bus {
	return s.Service.Bus()
}

func (s *ServiceBase) Logger() *zap.SugaredLogger {
	return s.Service.Logger()
}
