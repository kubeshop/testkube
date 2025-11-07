package services

import (
	"context"

	"github.com/kubeshop/testkube/cmd/api-server/commons"
	deprecatedapiv1 "github.com/kubeshop/testkube/internal/app/api/deprecatedv1"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/scheduler"
)

type DeprecatedSystem struct {
	Clients      commons.DeprecatedClients
	Repositories commons.DeprecatedRepositories
	Scheduler    *scheduler.Scheduler
	JobExecutor  client.Executor
	API          *deprecatedapiv1.DeprecatedTestkubeAPI
	StreamLogs   func(ctx context.Context, executionID string) (chan output.Output, error)
}
