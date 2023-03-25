package graph

import (
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"go.uber.org/zap"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Bus    bus.Bus
	Log    *zap.SugaredLogger
	Client *executorsclientv1.ExecutorsClient
}
