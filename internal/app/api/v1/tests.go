package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"

	"k8s.io/apimachinery/pkg/api/errors"
)

// GetTestHandler for getting test object
func (s TestKubeAPI) GetTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		crTest, err := s.TestsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		test := testsmapper.MapCRToAPI(*crTest)

		return c.JSON(test)
	}
}

// ListTestsHandler for getting list of all available tests
func (s TestKubeAPI) ListTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		s.Log.Debug("Getting scripts list")
		namespace := c.Query("namespace", "testkube")
		crTests, err := s.TestsClient.List(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		tests := testsmapper.MapTestListKubeToAPI(*crTests)

		return c.JSON(tests)
	}
}

func (s TestKubeAPI) ExecuteTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")

		s.Log.Debugw("getting script ", "name", name)
		crTest, err := s.TestsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		test := testsmapper.MapCRToAPI(*crTest)

		s.Log.Debugw("executing script", "name", name)

		c.JSON(test)

		return nil
	}
}
