package server

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/kubtest/pkg/executor/repository/result"
	"github.com/kubeshop/kubtest/pkg/runner"
	"github.com/kubeshop/kubtest/pkg/server"
	"github.com/kubeshop/kubtest/pkg/worker"
)

// ConcurrentExecutions per node
const ConcurrentExecutions = 4

// NewExecutor returns new CypressExecutor instance
func NewExecutor(resultRepository result.Repository, runner runner.Runner) Executor {
	var httpConfig server.Config
	envconfig.Process("EXECUTOR", &httpConfig)

	e := Executor{
		HTTPServer: server.NewServer(httpConfig),
		Repository: resultRepository,
		Worker:     worker.NewWorker(resultRepository, runner),
	}

	return e
}

type Executor struct {
	server.HTTPServer
	Repository result.Repository
	Worker     worker.Worker
}

// Init initialize ExecutorAPI server
func (e *Executor) Init() *Executor {

	executions := e.Routes.Group("/executions")

	// add standard start/get handlers from kubtest executor server library
	// they will push and get from worker queue storage
	executions.Post("/", e.StartExecution())
	executions.Get("/:id", e.GetExecution())

	return e
}

func (e Executor) Run() error {
	// get executions channel
	executionsQueue := e.Worker.PullExecutions()
	// pass channel to worker
	e.Worker.Run(executionsQueue)

	// run server (blocks process/returns error)
	return e.HTTPServer.Run()
}
