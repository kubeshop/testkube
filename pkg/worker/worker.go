package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/kubeshop/kubtest/pkg/executor/repository/result"
	"github.com/kubeshop/kubtest/pkg/log"
	"github.com/kubeshop/kubtest/pkg/runner"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const EmptyQueueWaitTime = 2 * time.Second
const WorkerQueueBufferSize = 10000

// NewWorker returns new worker instance with data repository and runner
func NewWorker(resultsRepository result.Repository, runner runner.Runner) Worker {
	return Worker{
		Concurrency: 4,
		BufferSize:  WorkerQueueBufferSize,
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

func (w *Worker) PullExecution() (execution kubtest.Execution, err error) {
	execution, err = w.Repository.QueuePull(context.Background())
	if err != nil {
		return execution, err
	}
	return
}

// PullExecutions gets executions from queue - returns executions channel
func (w *Worker) PullExecutions() chan kubtest.Execution {
	executionChan := make(chan kubtest.Execution, w.BufferSize)

	go func(executionChan chan kubtest.Execution) {
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

func (w *Worker) Run(executionChan chan kubtest.Execution) {
	for i := 0; i < w.Concurrency; i++ {
		go func(executionChan chan kubtest.Execution) {
			ctx := context.Background()
			for {
				e := <-executionChan
				l := w.Log.With("executionID", e.Id)
				l.Infow("Got script to run")

				e, err := w.RunExecution(ctx, e)
				if err != nil {
					l.Errorw("execution error", "error", err, "execution", e)
				} else {
					l.Infow("execution completed", "status", e.Result.Status)
				}

			}
		}(executionChan)
	}
}

func (w *Worker) RunExecution(ctx context.Context, e kubtest.Execution) (kubtest.Execution, error) {
	e.Result.Start()
	l := w.Log.With("executionID", e.Id, "startTime", e.Result.StartTime.String())

	// save start time
	if werr := w.Repository.UpdateResult(ctx, e.Id, *e.Result); werr != nil {
		return e, werr
	}

	l.Infow("script started", "status", e.Result.Status)
	result := w.Runner.Run(e)
	l.Infow("got result from runner", "result", result, "runner", fmt.Sprintf("%T", w.Runner))
	e.Result = &result

	var err error
	if result.ErrorMessage != "" {
		e.Result.Error()
		err = fmt.Errorf("execution error: %s", result.ErrorMessage)
	} else {
		e.Result.Success()
	}

	e.Result.Stop()

	// save end time
	if werr := w.Repository.UpdateResult(ctx, e.Id, *e.Result); werr != nil {
		return e, werr
	}
	l.Infow("script ended", "status", e.Result.Status, "endTime", e.Result.EndTime.String())

	return e, err
}
