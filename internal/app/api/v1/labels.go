package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (s TestkubeAPI) ListLabelsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		labels := make(map[string][]string)
		sources := append(*s.LabelSources, s.TestsClient, s.TestsSuitesClient)

		for _, source := range sources {
			nextLabels, err := source.ListLabels()
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("failed to list labels: %w", err))
			}

			for key, testValues := range nextLabels {
				valuesMap := make(map[string]struct{})
				if values, ok := labels[key]; ok {
					for _, v := range values {
						valuesMap[v] = struct{}{}
					}
				}

				for _, label := range testValues {
					if _, ok := valuesMap[label]; !ok {
						labels[key] = append(labels[key], label)
						valuesMap[label] = struct{}{}
					}
				}
			}
		}

		return c.JSON(labels)
	}
}
