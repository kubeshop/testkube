package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// GetConfigsHandler returns configuration
func (s TestkubeAPI) GetConfigsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		config, err := s.ConfigMap.Get(ctx)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("unable to get config: %w", err))
		}
		return c.JSON(config)
	}
}

// UpdateConfigsHandler update configuration handler
func (s TestkubeAPI) UpdateConfigsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		config, err := s.ConfigMap.Get(ctx)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("unable to get config: %w", err))
		}

		var request testkube.Config
		err = c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("config request body invalid: %w", err))
		}
		s.Log.Warnw("#######", "request", config)
		config.EnableTelemetry = request.EnableTelemetry
		s.Log.Warnw("#######", "request", config)
		err = s.ConfigMap.Upsert(ctx, config)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("unable to update config: %w", err))
		}
		return c.JSON(config)
	}
}
