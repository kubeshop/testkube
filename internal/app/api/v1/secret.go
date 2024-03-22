package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// ListSecretsHandler list secrets and keys
func (s TestkubeAPI) ListSecretsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list secrets"

		all, err := strconv.ParseBool(c.Query("all", "false"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse all parameter: %s", errPrefix, err))
		}

		list, err := s.SecretClient.List(all, c.Query("namespace"))
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
