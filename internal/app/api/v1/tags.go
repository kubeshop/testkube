package v1

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (s TestKubeAPI) ListTagsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		tags, err := s.ExecutionResults.GetTags(ctx)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		return c.JSON(tags)
	}
}
