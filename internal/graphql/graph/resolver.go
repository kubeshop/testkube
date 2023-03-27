//go:generate go run github.com/99designs/gqlgen generate

package graph

import (
	"go.uber.org/zap"

	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/event/bus"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Bus    bus.Bus
	Log    *zap.SugaredLogger
	Client *executorsclientv1.ExecutorsClient
}
