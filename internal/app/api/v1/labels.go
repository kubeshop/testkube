package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (s TestkubeAPI) ListLabelsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		testSuitesLabels, err := s.TestsSuitesClient.ListLabels()
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		labels, err := s.TestsClient.ListLabels()
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		for key, testValues := range testSuitesLabels {
			if values, ok := labels[key]; !ok {
				labels[key] = testValues
			} else {
				valuesMap := map[string]struct{}{}
				for _, v := range values {
					valuesMap[v] = struct{}{}
				}

				testValuesMap := map[string]struct{}{}
				for _, v := range testValues {
					testValuesMap[v] = struct{}{}
				}

				for k := range testValuesMap {
					if _, ok := valuesMap[k]; !ok {
						labels[key] = append(labels[key], k)
					}
				}
			}
		}

		return c.JSON(labels)
	}
}
