package v1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type LabelSource interface {
	ListLabels(ctx context.Context) (map[string][]string, error)
}

type extendedLabelSource interface {
	ListLabels(ctx context.Context, environmentId string) (map[string][]string, error)
}

type simpleLabelSource struct {
	source        extendedLabelSource
	environmentId string
}

func (s simpleLabelSource) ListLabels(ctx context.Context) (map[string][]string, error) {
	return s.source.ListLabels(ctx, s.environmentId)
}

func getClientLabelSource(source extendedLabelSource, environmentId string) LabelSource {
	return &simpleLabelSource{source: source, environmentId: environmentId}
}

func (s *TestkubeAPI) getEnvironmentId() string {
	if s.proContext != nil {
		return s.proContext.EnvID
	}
	return ""
}

func (s *TestkubeAPI) ListLabelsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		labels := make(map[string][]string)
		sources := []LabelSource{
			getClientLabelSource(s.TestWorkflowsClient, s.getEnvironmentId()),
			getClientLabelSource(s.TestWorkflowTemplatesClient, s.getEnvironmentId()),
		}

		for _, source := range sources {
			nextLabels, err := source.ListLabels(c.Context())
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
		id := c.Params("id")

		tags, err := s.TestWorkflowResults.GetExecutionTags(c.Context(), id)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		return c.JSON(tags)
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
