package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/utils"
)

func (s TestkubeAPI) ListTagsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")
		testsTags, err := s.TestsClient.ListTags(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		scriptsTags, err := s.ScriptsClient.ListTags(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		tags := append(testsTags, scriptsTags...)

		tags = utils.RemoveDuplicates(tags)

		return c.JSON(tags)
	}
}
