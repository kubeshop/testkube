package postman

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/kubetest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/postman/worker"
	"github.com/kubeshop/kubetest/internal/pkg/server"
	"github.com/kubeshop/kubetest/pkg/api/kubetest"
)

// ConcurrentExecutions per node
const ConcurrentExecutions = 4

// NewPostmanExecutor returns new PostmanExecutor instance
func NewPostmanExecutor(resultRepository result.Repository) PostmanExecutor {
	var httpConfig server.Config
	envconfig.Process("POSTMANEXECUTOR", &httpConfig)

	e := PostmanExecutor{
		HTTPServer: server.NewServer(httpConfig),
		Repository: resultRepository,
		Worker:     worker.NewWorker(resultRepository),
	}

	return e
}

type PostmanExecutor struct {
	server.HTTPServer
	Repository result.Repository
	Worker     worker.Worker
}

func (p *PostmanExecutor) Init() {
	executions := p.Routes.Group("/executions")
	executions.Post("/", p.StartExecution())
	executions.Get("/:id", p.GetExecution())
}

func (p *PostmanExecutor) StartExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request kubetest.ExecutionRequest
		err := json.Unmarshal(c.Body(), &request)
		if err != nil {
			return p.Error(c, 400, err)
		}

		execution := kubetest.NewExecution(string(request.Metadata), request.Params)
		err = p.Repository.Insert(context.Background(), execution)
		if err != nil {
			return p.Error(c, 400, err)

		}

		p.Log.Infow("starting new execution", "execution", execution)
		c.Response().Header.SetStatusCode(201)
		return c.JSON(execution)
	}
}

func (p PostmanExecutor) GetExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		execution, err := p.Repository.Get(context.Background(), c.Params("id"))
		if err != nil {
			return p.Error(c, 400, err)
		}

		return c.JSON(execution)
	}
}

func (p PostmanExecutor) Run() error {
	executionsQueue := p.Worker.PullExecutions()
	p.Worker.Run(executionsQueue)

	return p.HTTPServer.Run()
}
