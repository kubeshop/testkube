package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	scriptsMapper "github.com/kubeshop/testkube/pkg/mapper/scripts"

	"k8s.io/apimachinery/pkg/api/errors"
)

// ListScripts for getting list of all available scripts
func (s testkubeAPI) GetTest() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		crScript, err := s.ScriptsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptCRToAPI(*crScript)

		return c.JSON(scripts)
	}
}

// ListScripts for getting list of all available scripts
func (s testkubeAPI) ListTests() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")
		crScripts, err := s.ScriptsClient.List(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptListKubeToAPI(*crScripts)

		return c.JSON(scripts)
	}
}
