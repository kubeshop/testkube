package client

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/jobs"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
)

func NewJobExecutor(repo result.Repository) (client JobExecutor, err error) {
	jobClient, err := jobs.NewJobClient()
	if err != nil {
		return client, fmt.Errorf("can't get k8s jobs client: %w", err)
	}

	return JobExecutor{
		Client:     jobClient,
		Repository: repo,
		Log:        log.DefaultLogger,
	}, nil
}

type JobExecutor struct {
	Client     *jobs.JobClient
	Repository result.Repository
	Log        *zap.SugaredLogger
}

// Watch will get valid execution after async Execute, execution will be returned when success or error occurs
// Worker should set valid state for success or error after script completion
// TODO add timeout
func (c JobExecutor) Watch(id string) (events chan ResultEvent) {
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

func (c JobExecutor) Get(id string) (execution testkube.ExecutionResult, err error) {
	exec, err := c.Repository.Get(context.Background(), id)
	if err != nil {
		return testkube.ExecutionResult{}, err
	}
	return *exec.ExecutionResult, nil
}

// Logs returns job logs using kubernetes api
func (c JobExecutor) Logs(id string) (out chan output.Output, err error) {
	out = make(chan output.Output)
	logs := make(chan []byte)

	go func() {
		defer func() {
			c.Log.Debug("closing JobExecutor.Logs out log")
			close(out)
		}()

		if err := c.Client.TailJobLogs(id, logs); err != nil {
			out <- output.NewOutputError(err)
			return
		}

		for l := range logs {
			entry, err := output.GetLogEntry(l)
			if err != nil {
				out <- output.NewOutputError(err)
				return
			}
			out <- entry
		}
	}()

	return
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c JobExecutor) Execute(execution testkube.Execution, options ExecuteOptions) (result testkube.ExecutionResult, err error) {
	return c.Client.LaunchK8sJob(options.ExecutorSpec.Image, c.Repository, execution, options.HasSecrets)
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started synchronously client will be blocked
func (c JobExecutor) ExecuteSync(execution testkube.Execution, options ExecuteOptions) (result testkube.ExecutionResult, err error) {
	return c.Client.LaunchK8sJobSync(options.ExecutorSpec.Image, c.Repository, execution, options.HasSecrets)
}

func (c JobExecutor) Abort(id string) error {
	c.Client.AbortK8sJob(id)
	return nil
}
