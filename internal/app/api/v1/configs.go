package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// GetConfigsHandler returns configuration
func (s TestkubeAPI) GetConfigsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		config, err := s.ConfigMap.Get(ctx)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("failed to get config: db found no config: %w", err))
			}
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("failed to get config: %w", err))
		}
		return c.JSON(config)
	}
}

// UpdateConfigsHandler update configuration handler
func (s TestkubeAPI) UpdateConfigsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		errPrefix := "failed to update config"

		config, err := s.ConfigMap.Get(ctx)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: db found no config: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: unable to get config: %w", errPrefix, err))
		}

		var request testkube.Config
		err = c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: config request body invalid: %w", errPrefix, err))
		}
		s.Log.Warnw("#######", "request", config)
		config.EnableTelemetry = request.EnableTelemetry
		s.Log.Warnw("#######", "request", config)
		_, err = s.ConfigMap.Upsert(ctx, config)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: db could not update config: %w", errPrefix, err))
		}
		return c.JSON(config)
	}
}
