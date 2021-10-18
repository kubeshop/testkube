package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/repository/result"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/runner"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const EmptyQueueWaitTime = 2 * time.Second
const Concurrency = 4

// NewWorker returns new worker instance with data repository and runner
func NewWorker(resultsRepository result.Repository, runner runner.Runner) Worker {
	return Worker{
		Concurrency: Concurrency,
		BufferSize:  Concurrency,
		Repository:  resultsRepository,
		// TODO implement runner for new executor
		Runner: runner,
		Log:    log.DefaultLogger,
	}
}

type Worker struct {
	Concurrency int
	BufferSize  int
	Repository  result.Repository
	Runner      runner.Runner
	Log         *zap.SugaredLogger
}

func (w *Worker) PullExecution() (execution testkube.Execution, err error) {
	execution, err = w.Repository.QueuePull(context.Background())
	if err != nil {
		return execution, err
	}
	return
}

// PullExecutions gets executions from queue - returns executions channel
func (w *Worker) PullExecutions() chan testkube.Execution {
	executionChan := make(chan testkube.Execution, w.BufferSize)

	go func(executionChan chan testkube.Execution) {
		w.Log.Info("Watching queue start")
		for {
			execution, err := w.PullExecution()
			if err != nil {
				if err == mongo.ErrNoDocuments {
					w.Log.Debug("no records found in queue to process")
					time.Sleep(EmptyQueueWaitTime)
					continue
				}
				w.Log.Errorw("pull execution error", "error", err)
				continue
			}

			executionChan <- execution
		}
	}(executionChan)

	return executionChan
}

func (w *Worker) Run(executionChan chan testkube.Execution) {
	for i := 0; i < w.Concurrency; i++ {
		go func(executionChan chan testkube.Execution) {
			ctx := context.Background()
			for {
				e := <-executionChan
				l := w.Log.With("executionID", e.Id)
				l.Infow("Got script to run")

				e, err := w.RunExecution(ctx, e)
				if err != nil {
					l.Errorw("execution error", "error", err, "execution", e)
				} else {
					l.Infow("execution completed", "status", e.ExecutionResult.Status)
				}

			}
		}(executionChan)
	}
}

func (w *Worker) RunExecution(ctx context.Context, e testkube.Execution) (testkube.Execution, error) {
	startTime := time.Now()
	l := w.Log.With("executionID", e.Id)
	e.ExecutionResult.StartTime = startTime
	// save start time
	if werr := w.Repository.UpdateResult(ctx, e.Id, *e.ExecutionResult); werr != nil {
		return e, werr
	}

	l.Infow("script started", "status", e.ExecutionResult.Status)
	result := w.Runner.Run(e)
	result.StartTime = startTime
	result.EndTime = time.Now()
	l.Infow("got result from runner", "result", result, "runner", fmt.Sprintf("%T", w.Runner))
	e.ExecutionResult = &result

	var err error
	if result.ErrorMessage != "" {
		e.ExecutionResult.Error()
		err = fmt.Errorf("execution error: %s", result.ErrorMessage)
	} else {
		e.ExecutionResult.Success()
	}

	// save end time
	if werr := w.Repository.UpdateResult(ctx, e.Id, *e.ExecutionResult); werr != nil {
		return e, werr
	}
	l.Infow("script ended", "status", e.ExecutionResult.Status, "startTime", e.ExecutionResult.StartTime.String(), "endTime", e.ExecutionResult.EndTime.String())

	return e, err
}
