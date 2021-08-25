package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubeshop/kubtest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/kubeshop/kubtest/pkg/log"
	"github.com/kubeshop/kubtest/pkg/runner/newman"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const EmptyQueueWaitTime = 2 * time.Second
const WorkerQueueBufferSize = 10000

func NewWorker(resultsRepository result.Repository) Worker {
	return Worker{
		Concurrency: 4,
		BufferSize:  WorkerQueueBufferSize,
		Repository:  resultsRepository,
		Runner:      &newman.Runner{},
		Log:         log.DefaultLogger,
	}
}

type Worker struct {
	Concurrency int
	BufferSize  int
	Repository  result.Repository
	Runner      *newman.Runner
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
		w.Log.Info("starting queue watcher")
		for {
			execution, err := w.PullExecution()
			l := w.Log.With("executionID", execution.Id, "items", len(executionChan), "status", execution.Status)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					w.Log.Debug("no records found in queue to process")
					time.Sleep(EmptyQueueWaitTime)
					continue
				}
				w.Log.Errorw("pull execution error", "error", err)
				continue
			}

			l.Debug("pushing to executionChan")
			executionChan <- execution
			l.Debug("pushed to executionChan")
		}
	}(executionChan)

	return executionChan
}

func (w *Worker) Run(executionChan chan kubtest.Execution) {
	for i := 0; i < w.Concurrency; i++ {
		go func(executionChan chan kubtest.Execution, i int) {
			ctx := context.Background()
			for {
				e := <-executionChan
				l := w.Log.With("executionID", e.Id, "workerID", i)
				l.Infow("running execution")

				e, err := w.RunExecution(ctx, e)
				if err != nil {
					l.Errorw("execution error", "error", err, "execution", e)
				} else {
					l.Infow("execution completed", "status", e.Status)
				}

			}
		}(executionChan, i)
	}
}

func (w *Worker) RunExecution(ctx context.Context, e kubtest.Execution) (kubtest.Execution, error) {
	e.Start()
	// write start time to repo
	if werr := w.Repository.Update(ctx, e); werr != nil {
		return e, werr
	}

	result := w.Runner.Run(strings.NewReader(e.ScriptContent), e.Params)
	e.Result = &result

	var err error
	if result.ErrorMessage != "" {
		e.Error()
		err = fmt.Errorf("execution error: %s", result.ErrorMessage)
	} else {
		e.Success()
	}

	e.Stop()
	// we want always write even if there is error
	if werr := w.Repository.Update(ctx, e); werr != nil {
		return e, werr
	}

	return e, err
}
