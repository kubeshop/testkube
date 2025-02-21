package deprecatedv1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/app/api/apiutils"
)

const (
	// mediaTypeJSON is json media type
	mediaTypeJSON = "application/json"
	// mediaTypeYAML is yaml media type
	mediaTypeYAML = "text/yaml"
)

// Warn writes RFC-7807 json problem to response
func (s *DeprecatedTestkubeAPI) Warn(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	return apiutils.SendWarn(c, status, err, context...)
}

// Error writes RFC-7807 json problem to response
func (s *DeprecatedTestkubeAPI) Error(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	return apiutils.SendError(c, status, err, context...)
}

func (s *DeprecatedTestkubeAPI) NotImplemented(c *fiber.Ctx) error {
	return s.Error(c, http.StatusNotImplemented, errors.New("not implemented yet"))
}

func (s *DeprecatedTestkubeAPI) BadGateway(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *DeprecatedTestkubeAPI) InternalError(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *DeprecatedTestkubeAPI) BadRequest(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *DeprecatedTestkubeAPI) NotFound(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *DeprecatedTestkubeAPI) ClientError(c *fiber.Ctx, prefix string, err error) error {
	if apiutils.IsNotFound(err) {
		return s.NotFound(c, prefix, "client not found", err)
	}
	return s.BadGateway(c, prefix, "client problem", err)
}
