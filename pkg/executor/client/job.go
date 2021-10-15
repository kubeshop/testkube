package client

import (
	"fmt"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/jobs"
)

func NewJobExecutorClient(repo result.Repository) (client JobExecutorClient, err error) {
	jobClient, err := jobs.NewJobClient()
	if err != nil {
		return client, fmt.Errorf("can't get k8s jobs client: %w", err)
	}

	return JobExecutorClient{
		Client: jobClient,
	}, nil
}

type JobExecutorClient struct {
	Client     *jobs.JobClient
	Repository result.Repository
}

// Watch will get valid execution after async Execute, execution will be returned when success or error occurs
// Worker should set valid state for success or error after script completion
// TODO add timeout
func (c JobExecutorClient) Watch(id string) (events chan ResultEvent) {
	events = make(chan ResultEvent)

	go func() {
		ticker := time.NewTicker(WatchInterval)
		for range ticker.C {
			result, err := c.Get(id)

			events <- ResultEvent{
				Result: result,
				Error:  err,
			}

			if err != nil || result.IsCompleted() {
				close(events)
				return
			}
		}

	}()

	return events
}

func (c JobExecutorClient) Get(id string) (execution testkube.ExecutionResult, err error) {
	// TODO Get Logs ? Update Execution
	return
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c JobExecutorClient) Execute(options ExecuteOptions) (result testkube.ExecutionResult, err error) {
	// TODO move to mapper
	execution := testkube.NewExecutionWithID(options.ID, options.ScriptSpec.Type_, options.ScriptSpec.Name)
	execution.ScriptContent = options.ScriptSpec.Content
	execution.Repository = (*testkube.Repository)(options.ScriptSpec.Repository)
	execution.Params = options.Request.Params

	return c.Client.LaunchK8sJob( options.ExecutorSpec.Image, c.Repository, execution)
}

func (c JobExecutorClient) Abort(id string) error {
	c.Client.AbortK8sJob(id)
	return nil
}
