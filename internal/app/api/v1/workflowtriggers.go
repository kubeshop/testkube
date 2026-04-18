package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/workflowtriggerclient"
)

// namePattern matches RFC 1123 subdomain. Shared convention with TestTrigger
// and webhook handlers; validated before the request hits the K8s apiserver.
var workflowTriggerNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

// ListWorkflowTriggersHandler returns all WorkflowTrigger v2 resources in the namespace.
// Pagination and textSearch are intentionally omitted until the underlying client
// supports them — controller-runtime's List has no native offset/limit/search.
// Clients filter via selector (label match) or client-side.
func (s *TestkubeAPI) ListWorkflowTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")

		triggers, err := s.WorkflowTriggersClient.List(c.Context(), s.getEnvironmentId(), workflowtriggerclient.ListOptions{
			Selector: selector,
		}, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("failed to list workflow triggers: %w", err))
		}
		return c.JSON(triggers)
	}
}

// validateWorkflowTriggerSpec rejects invalid values that the mapper would
// otherwise silently drop (e.g. unparseable Run.Delay). Without this, the HTTP
// response echoes the user's bad value while the stored CRD has a nil/empty one.
func validateWorkflowTriggerSpec(trigger *testkube.WorkflowTrigger) error {
	if trigger.Run.Delay != "" {
		if _, err := time.ParseDuration(trigger.Run.Delay); err != nil {
			return fmt.Errorf("invalid run.delay %q: %w", trigger.Run.Delay, err)
		}
	}
	return nil
}

// GetWorkflowTriggerHandler returns a single WorkflowTrigger by name.
func (s *TestkubeAPI) GetWorkflowTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get workflow trigger %s", name)

		trigger, err := s.WorkflowTriggersClient.Get(c.Context(), s.getEnvironmentId(), name, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %w", errPrefix, err))
		}
		return c.JSON(trigger)
	}
}

// CreateWorkflowTriggerHandler creates a new WorkflowTrigger from the JSON request body.
func (s *TestkubeAPI) CreateWorkflowTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create workflow trigger"

		var trigger testkube.WorkflowTrigger
		if err := json.Unmarshal(c.Body(), &trigger); err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
		}
		if trigger.Namespace == "" {
			trigger.Namespace = s.Namespace
		}
		if trigger.Name == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: name is required", errPrefix))
		}
		if !workflowTriggerNamePattern.MatchString(trigger.Name) {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: name %q is not a valid RFC 1123 subdomain", errPrefix, trigger.Name))
		}
		if err := validateWorkflowTriggerSpec(&trigger); err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: %w", errPrefix, err))
		}

		if err := s.WorkflowTriggersClient.Create(c.Context(), s.getEnvironmentId(), trigger); err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s %s: %w", errPrefix, trigger.Name, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(trigger)
	}
}

// UpdateWorkflowTriggerHandler replaces an existing WorkflowTrigger with the JSON request body.
func (s *TestkubeAPI) UpdateWorkflowTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to update workflow trigger %s", name)

		var trigger testkube.WorkflowTrigger
		if err := json.Unmarshal(c.Body(), &trigger); err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
		}
		// Prevent accidental rename via body - path id is authoritative.
		trigger.Name = name
		if trigger.Name == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: name is required", errPrefix))
		}
		if !workflowTriggerNamePattern.MatchString(trigger.Name) {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: name %q is not a valid RFC 1123 subdomain", errPrefix, trigger.Name))
		}
		if trigger.Namespace == "" {
			trigger.Namespace = s.Namespace
		}
		if err := validateWorkflowTriggerSpec(&trigger); err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: %w", errPrefix, err))
		}

		if err := s.WorkflowTriggersClient.Update(c.Context(), s.getEnvironmentId(), trigger); err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %w", errPrefix, err))
		}
		return c.JSON(trigger)
	}
}

// DeleteWorkflowTriggerHandler deletes a single WorkflowTrigger by name.
func (s *TestkubeAPI) DeleteWorkflowTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete workflow trigger %s", name)

		if err := s.WorkflowTriggersClient.Delete(c.Context(), s.getEnvironmentId(), name, namespace); err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %w", errPrefix, err))
		}
		return c.SendStatus(http.StatusNoContent)
	}
}

// DeleteWorkflowTriggersHandler bulk-deletes WorkflowTriggers in a namespace
// optionally filtered by label selector.
func (s *TestkubeAPI) DeleteWorkflowTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")
		errPrefix := "failed to delete workflow triggers"

		if selector != "" {
			if _, err := s.WorkflowTriggersClient.DeleteByLabels(c.Context(), s.getEnvironmentId(), selector, namespace); err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %w", errPrefix, err))
			}
		} else {
			if _, err := s.WorkflowTriggersClient.DeleteAll(c.Context(), s.getEnvironmentId(), namespace); err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %w", errPrefix, err))
			}
		}
		return c.SendStatus(http.StatusNoContent)
	}
}
