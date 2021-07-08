package postman

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/kubetest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubetest/pkg/api/executor"
	"github.com/kubeshop/kubetest/pkg/runner"
	"github.com/kubeshop/kubetest/pkg/runner/newman"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConcurrentExecutions per node
const ConcurrentExecutions = 4

// NewPostmanExecutor returns new PostmanExecutor instance
func NewPostmanExecutor() PostmanExecutor {
	e := PostmanExecutor{
		Mux:    fiber.New(),
		Runner: &newman.Runner{},
	}

	return e
}

type PostmanExecutor struct {
	Mux        *fiber.App
	Runner     runner.Runner
	Repository result.Repository
}

func (p *PostmanExecutor) Init() {
	// v1 API
	v1 := p.Mux.Group("/v1")
	v1.Static("/api-docs", "./api/v1")

	executions := v1.Group("/executions")
	executions.Post("/", p.StartExecution())
	executions.Get("/:id", p.GetExecution())
}

func (p *PostmanExecutor) StartExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request ExecuteRequest
		err := json.Unmarshal(c.Body(), &request)
		if err != nil {
			return err
		}

		// TODO UUID instead of BSON?
		execution := executor.NewExecution(
			primitive.NewObjectID().Hex(),
			request.Name,
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
	return p.Mux.Listen(":8082")
}
