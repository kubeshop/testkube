package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func (s testkubeAPI) CreateExecutor() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.ExecutorCreateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		return c.JSON(request)
	}
}

func (s testkubeAPI) ListExecutors() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON([]testkube.ExecutorDetails{})
	}
}

func (s testkubeAPI) GetExecutor() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		return c.JSON(testkube.ExecutorDetails{
			Name: name,
		})
	}
}

func (s testkubeAPI) DeleteExecutor() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_ = c.Params("name")
		c.Context().SetStatusCode(204)
		return nil
	}
}
