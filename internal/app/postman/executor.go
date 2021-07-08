package postman

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/kubetest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubetest/pkg/runner"
	"github.com/kubeshop/kubetest/pkg/runner/newman"
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

		content := strings.NewReader(string(request.Metadata))
		result, err := p.Runner.Run(content)
		if err != nil {
			return err
		}

		p.Repository.Insert(context.Background(), result)
		return c.JSON(result)
	}
}

func (p PostmanExecutor) GetExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("Getting execution 👋!")
	}
}

func (p PostmanExecutor) Run() error {
	return p.Mux.Listen(":8082")
}
