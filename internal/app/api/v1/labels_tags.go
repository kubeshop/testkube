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

			deduplicateMap(nextLabels, labels)
		}

		return c.JSON(labels)
	}
}

func (s *TestkubeAPI) ListTagsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list execution tags"

		tags, err := s.TestWorkflowResults.GetExecutionTags(c.Context())
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		results := make(map[string][]string)
		deduplicateMap(tags, results)

		return c.JSON(results)
	}
}

func deduplicateMap(src, dst map[string][]string) {
	for key, testValues := range src {
		valuesMap := make(map[string]struct{})
		if values, ok := dst[key]; ok {
			for _, v := range values {
				valuesMap[v] = struct{}{}
			}
		}

		for _, testValue := range testValues {
			if _, ok := valuesMap[testValue]; !ok {
				dst[key] = append(dst[key], testValue)
				valuesMap[testValue] = struct{}{}
			}
		}
	}
}
