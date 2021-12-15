package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"

	"k8s.io/apimachinery/pkg/api/errors"
)

// GetTest for getting test object
func (s TestKubeAPI) GetTest() fiber.Handler {
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

// ListTests for getting list of all available tests
func (s TestKubeAPI) ListTests() fiber.Handler {
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
