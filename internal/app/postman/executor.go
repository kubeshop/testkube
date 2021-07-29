package postman

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/kubetest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/postman/worker"
	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	"github.com/kubeshop/kubetest/pkg/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// ConcurrentExecutions per node
const ConcurrentExecutions = 4

// NewPostmanExecutor returns new PostmanExecutor instance
func NewPostmanExecutor(resultRepository result.Repository) PostmanExecutor {
	e := PostmanExecutor{
		Mux:        fiber.New(),
		Repository: resultRepository,
		Worker:     worker.NewWorker(resultRepository),
		Log:        log.DefaultLogger,
	}

	return e
}

type PostmanExecutor struct {
	Mux        *fiber.App
	Repository result.Repository
	Worker     worker.Worker
	Log        *zap.SugaredLogger
}

func (p *PostmanExecutor) Init() {
	p.Mux.Get("/health", p.HealthEndpoint())
	// v1 API
	v1 := p.Mux.Group("/v1")
	v1.Static("/api-docs", "./api/v1")

	executions := v1.Group("/executions")
	executions.Post("/", p.StartExecution())
	executions.Get("/:id", p.GetExecution())
}

func (p *PostmanExecutor) StartExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request kubetest.ExecuteRequest
		err := json.Unmarshal(c.Body(), &request)
		if err != nil {
			return err
		}

		p.Log.Infow("new execution request", "request", request)

		// TODO consider UUID instead of BSON? or some simplier/shorter id!?
		//      ID also can be executor local, in kubetest we'll handle all IDs as strings
		execution := kubetest.NewExecution(
			primitive.NewObjectID().Hex(),
			string(request.Metadata),
		)
		err = p.Repository.Insert(context.Background(), execution)
		if err != nil {
			return err
		}

		c.Response().Header.SetStatusCode(201)
		return c.JSON(execution)
	}
}

func (p PostmanExecutor) GetExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		execution, err := p.Repository.Get(context.Background(), c.Params("id"))
		if err != nil {
			return err
		}

		return c.JSON(execution)
	}
}

func (p PostmanExecutor) Run() error {

	executionsQueue := p.Worker.PullExecutions()
	p.Worker.Run(executionsQueue)

	return p.Mux.Listen(":8082")
}
