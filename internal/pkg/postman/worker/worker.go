package worker

import (
	"context"
	"strings"

	"github.com/kubeshop/kubetest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubetest/pkg/api/executor"
	"github.com/kubeshop/kubetest/pkg/log"
	"github.com/kubeshop/kubetest/pkg/runner/newman"
	"go.uber.org/zap"
)

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

func (w Worker) PullExecution() (execution executor.Execution, err error) {
	execution, err = w.Repository.QueuePull(context.Background())
	if err != nil {
		return execution, err
	}
	execution.Start()
	return
}

func (w Worker) PullExecutions() chan executor.Execution {
	executionChan := make(chan executor.Execution, w.BufferSize)

	go func(executionChan chan executor.Execution) {
		w.Log.Info("Pulling data from queue")
		for {
			execution, err := w.PullExecution()
			if err != nil {
				w.Log.Errorw("pull execution error", "error", err)
				continue
			}

			executionChan <- execution
		}
	}(executionChan)

	return executionChan
}

func (w Worker) Run(executionChan chan executor.Execution) {
	for i := 0; i < w.Concurrency; i++ {
		go func(executionChan chan executor.Execution) {
			ctx := context.Background()
			for {
				e := <-executionChan
				w.Log.Info("Got script to run", "type", e.ScriptType, "content", e.ScriptContent, "name", e.Name, "id", e.Id)
				result, err := w.Runner.Run(strings.NewReader(e.ScriptContent))
				if err != nil {
					w.Log.Errorw("script execution error", "error", err)
					e.Error(err)
				}
				e.Output = result
				w.Repository.Update(ctx, e)

			}
		}(executionChan)
	}
}
