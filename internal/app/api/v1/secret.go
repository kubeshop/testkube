package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// ListSecretsHandler list secrets and keys
func (s TestkubeAPI) ListSecretsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list secrets"

		list, err := s.SecretClient.List()
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list secrets: %s", errPrefix, err))
		}

		results := make([]testkube.Secret, 0)
		for name, values := range list {
			keys := make([]string, 0)
			for value := range values {
				keys = append(keys, value)
			}

			results = append(results, testkube.Secret{Name: name, Keys: keys})
		}

		return c.JSON(results)
	}
}
