package postman

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/kubetest/pkg/newman"
)

// ConcurrentExecutions per node
const ConcurrentExecutions = 4

// NewPostmanExecutor returns new PostmanExecutor instance
func NewPostmanExecutor() PostmanExecutor {
	return PostmanExecutor{
		Mux: fiber.New(),
	}
}

type PostmanExecutor struct {
	Mux    *fiber.App
	Runner newman.Runner
}

func (p PostmanExecutor) Init() {
	// v1 API
	v1 := p.Mux.Group("/v1")
	v1.Static("/api-docs", "./api/v1")

	executions := v1.Group("/executions")
	executions.Post("/", p.StartExecution())
	executions.Post("/:id", p.GetExecution())
}

func (p PostmanExecutor) StartExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request ExecuteRequest
		c.BodyParser(request)
		content := strings.NewReader(request.Body)
		result, err := p.Runner.RunCollection(content)
		if err != nil {
			return err
		}
		c.JSON(result)
		return err
	}
}

func (p PostmanExecutor) GetExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("Getting execution 👋!")
	}
}

func (p PostmanExecutor) Run() error {
	return p.Mux.Listen(":8081")
}
