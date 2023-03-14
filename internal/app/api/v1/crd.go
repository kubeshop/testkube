package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (s TestkubeAPI) getCRDs(c *fiber.Ctx, data string, err error) error {
	if err != nil {
		return s.Error(c, http.StatusBadRequest, fmt.Errorf("could not build CRD: %w", err))
	}

	c.Context().SetContentType(mediaTypeYAML)
	return c.SendString(data)
}
