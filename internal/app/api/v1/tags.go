package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/utils"
)

func (s TestkubeAPI) ListTagsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")
		testsTags, err := s.TestsSuitesClient.ListTags(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		tagList, err := s.TestsClient.ListTags(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		tags := append(testsTags, tagList...)

		tags = utils.RemoveDuplicates(tags)

		return c.JSON(tags)
	}
}
