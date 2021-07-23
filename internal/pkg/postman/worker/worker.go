package worker

import (
	"context"
	"strings"
	"time"

	"github.com/kubeshop/kubetest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	"github.com/kubeshop/kubetest/pkg/log"
	"github.com/kubeshop/kubetest/pkg/runner/newman"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const EmptyQueueWaitTime = 2 * time.Second

func NewWorker(resultsRepository result.Repository) Worker {
	return Worker{
		Concurrency: 4,
		BufferSize:  10000,
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

func (w *Worker) PullExecution() (execution kubetest.Execution, err error) {
	execution, err = w.Repository.QueuePull(context.Background())
	if err != nil {
		return execution, err
	}
	return
}

func (w *Worker) PullExecutions() chan kubetest.Execution {
	executionChan := make(chan kubetest.Execution, w.BufferSize)

	go func(executionChan chan kubetest.Execution) {
		w.Log.Info("Watching queue start")
		for {
			execution, err := w.PullExecution()
			if err != nil {
				if err == mongo.ErrNoDocuments {
					w.Log.Debug("no records found in queue to process")
					// TODO - to not kill mongo - consider some exp function
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

func (w *Worker) Run(executionChan chan kubetest.Execution) {
	for i := 0; i < w.Concurrency; i++ {
		go func(executionChan chan kubetest.Execution) {
			ctx := context.Background()
			for {
				e := <-executionChan
				l := w.Log.With("type", e.ScriptType, "executionID", e.Id)
				l.Infow("Got script to run")

				e, err := w.RunExecution(ctx, e)
				if err != nil {
					l.Errorw("execution error", "error", err, "execution", e)
				} else {
					l.Infow("execution completed", "status", e.Status)
				}

			}
		}(executionChan)
	}
}

func (w *Worker) RunExecution(ctx context.Context, e kubetest.Execution) (kubetest.Execution, error) {
	e.Start()
	result, err := w.Runner.Run(strings.NewReader(e.ScriptContent))
	e.Stop()
	e.Output = result

	if err != nil {
		e.Error(err)
	} else {
		e.Success()
	}

	return e, w.Repository.Update(ctx, e)
}
