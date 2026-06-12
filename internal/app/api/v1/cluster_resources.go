package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// ListClusterResourcesHandler returns every GVK the cluster exposes, tagged
// with whether the agent's ServiceAccount can watch it. Pass ?watchable=true
// to drop entries the agent cannot list+watch.
func (s *TestkubeAPI) ListClusterResourcesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if s.ClusterDiscoverer == nil {
			return s.Error(c, http.StatusNotImplemented, fmt.Errorf("cluster discovery is not configured on this instance"))
		}
		resources, err := s.ClusterDiscoverer.List(c.Context())
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("cluster discovery: %w", err))
		}
		if c.QueryBool("watchable") {
			watchable := resources[:0]
			for _, r := range resources {
				if r.CanWatch {
					watchable = append(watchable, r)
				}
			}
			resources = watchable
		}
		return c.JSON(resources)
	}
}
