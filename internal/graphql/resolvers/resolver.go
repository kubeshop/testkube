//go:generate go run github.com/99designs/gqlgen generate

package resolvers

import (
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"go.uber.org/zap"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type ResolverData interface {
	Bus() bus.Bus
	Logger() *zap.SugaredLogger
	ExecutorsClient() *executorsclientv1.ExecutorsClient
}

type Resolver struct {
	BusInstance             bus.Bus
	LoggerInstance          *zap.SugaredLogger
	ExecutorsClientInstance *executorsclientv1.ExecutorsClient
}

func (r *Resolver) Bus() bus.Bus {
	return r.BusInstance
}

func (r *Resolver) Logger() *zap.SugaredLogger {
	return r.LoggerInstance
}

func (r *Resolver) ExecutorsClient() *executorsclientv1.ExecutorsClient {
	return r.ExecutorsClientInstance
}
