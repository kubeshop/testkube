package v1

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/ai"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func (s TestkubeAPI) AnalyzeTestExecution() fiber.Handler {
	ai := ai.NewOpenAI(os.Getenv("OPENAI_KEY"))
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		execution, err := s.ExecutionResults.Get(ctx, c.Params("executionID"))
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't get execution details: %w", err))
		}

		if *execution.ExecutionResult.Status != *testkube.ExecutionStatusFailed {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("execution status is not failed, I can analyze only failed executions"))
		}

		resp, err := ai.AnalyzeTestExecution(ctx, execution)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("client could not analyze test execution: %w", err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(testkube.AiAnalysisResponse{
			Message: resp,
		})
	}
}
