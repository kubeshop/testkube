package v1

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/yaml"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/keymap/triggers"
	triggerskeymapmapper "github.com/kubeshop/testkube/pkg/mapper/keymap/triggers"
	testtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/testtriggers"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/utils"
)

const testTriggerMaxNameLength = 57

// CreateTestTriggerHandler is a handler for creating test trigger objects
func (s *TestkubeAPI) CreateTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create test trigger"
		var testTrigger testtriggersv1.TestTrigger
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			testTriggerSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(testTriggerSpec), len(testTriggerSpec))
			if err := decoder.Decode(&testTrigger); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}
		} else {
			var request testkube.TestTriggerUpsertRequest
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}

			testTrigger = testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(request)
			// default namespace if not defined in upsert request
			if testTrigger.Namespace == "" {
				testTrigger.Namespace = s.Namespace
			}
			// default trigger name if not defined in upsert request
			if testTrigger.Name == "" {
				testTrigger.Name = generateTestTriggerName(&testTrigger)
			}

			if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
				data, err := crd.GenerateYAML(crd.TemplateTestTrigger, []testkube.TestTrigger{testtriggersmapper.MapCRDToAPI(&testTrigger)})
				return apiutils.SendLegacyCRDs(c, data, err)
			}
		}

		errPrefix = errPrefix + " " + testTrigger.Name

		s.Log.Infow("creating test trigger", "testTrigger", testTrigger)

		// Convert CRD to API object for the new interface
		apiTrigger := testtriggersmapper.MapCRDToAPI(&testTrigger)

		err := s.TestTriggersClient.Create(c.Context(), s.getEnvironmentId(), apiTrigger)

		s.Metrics.IncCreateTestTrigger(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create test trigger: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(apiTrigger)
	}
}

// UpdateTestTriggerHandler is a handler for updates an existing TestTrigger CRD based on TestTrigger content
func (s *TestkubeAPI) UpdateTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to update test trigger"
		var request testkube.TestTriggerUpsertRequest
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			var testTrigger testtriggersv1.TestTrigger
			testTriggerSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(testTriggerSpec), len(testTriggerSpec))
			if err := decoder.Decode(&testTrigger); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}

			request = testtriggersmapper.MapTestTriggerCRDToTestTriggerUpsertRequest(testTrigger)
		} else {
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}

		}

		namespace := s.Namespace
		if request.Namespace != "" {
			namespace = request.Namespace
		}
		errPrefix = errPrefix + " " + request.Name

		// we need to get resource first
		apiTrigger, err := s.TestTriggersClient.Get(c.Context(), s.getEnvironmentId(), request.Name, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test trigger: %w", errPrefix, err))
		}

		// Update the trigger with new values from request
		apiTrigger.Name = request.Name
		apiTrigger.Namespace = namespace
		if request.Labels != nil {
			apiTrigger.Labels = request.Labels
		}
		if request.Annotations != nil {
			apiTrigger.Annotations = request.Annotations
		}

		// Update individual fields from the request
		if request.Resource != nil {
			apiTrigger.Resource = request.Resource
		}
		if request.ResourceSelector != nil {
			apiTrigger.ResourceSelector = request.ResourceSelector
		}
		if request.Event != "" {
			apiTrigger.Event = request.Event
		}
		if request.ConditionSpec != nil {
			apiTrigger.ConditionSpec = request.ConditionSpec
		}
		if request.ProbeSpec != nil {
			apiTrigger.ProbeSpec = request.ProbeSpec
		}
		if request.ContentSelector != nil {
			apiTrigger.ContentSelector = request.ContentSelector
		}
		if request.Action != nil {
			apiTrigger.Action = request.Action
		}
		if request.ActionParameters != nil {
			apiTrigger.ActionParameters = request.ActionParameters
		}
		if request.Execution != nil {
			apiTrigger.Execution = request.Execution
		}
		if request.TestSelector != nil {
			apiTrigger.TestSelector = request.TestSelector
		}
		if request.ConcurrencyPolicy != nil {
			apiTrigger.ConcurrencyPolicy = request.ConcurrencyPolicy
		}
		apiTrigger.Disabled = request.Disabled

		err = s.TestTriggersClient.Update(c.Context(), s.getEnvironmentId(), *apiTrigger)
		s.Metrics.IncUpdateTestTrigger(err)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update test trigger: %w", errPrefix, err))
		}

		return c.JSON(apiTrigger)
	}
}

// BulkUpdateTestTriggersHandler is a handler for bulk updates an existing TestTrigger CRDs based on array of TestTrigger content
func (s *TestkubeAPI) BulkUpdateTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to bulk update test triggers"

		var request []testkube.TestTriggerUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse request: %w", errPrefix, err))
		}

		namespaces := make(map[string]struct{}, 0)
		for _, upsertRequest := range request {
			namespace := s.Namespace
			if upsertRequest.Namespace != "" {
				namespace = upsertRequest.Namespace
			}

			namespaces[namespace] = struct{}{}
		}

		for namespace := range namespaces {
			_, err = s.TestTriggersClient.DeleteAll(c.Context(), s.getEnvironmentId(), namespace)
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: error cleaning triggers before reapply", errPrefix))
			}
		}

		s.Metrics.IncBulkDeleteTestTrigger(nil)

		testTriggers := make([]testkube.TestTrigger, 0, len(request))

		for _, upsertRequest := range request {
			crdTestTrigger := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(upsertRequest)
			// default trigger name if not defined in upsert request
			if crdTestTrigger.Name == "" {
				crdTestTrigger.Name = generateTestTriggerName(&crdTestTrigger)
			}

			// Convert CRD to API object for the new interface
			apiTrigger := testtriggersmapper.MapCRDToAPI(&crdTestTrigger)

			err = s.TestTriggersClient.Create(c.Context(), s.getEnvironmentId(), apiTrigger)

			s.Metrics.IncCreateTestTrigger(err)

			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: error reapplying triggers after clean", errPrefix))
			}

			testTriggers = append(testTriggers, apiTrigger)
		}

		s.Metrics.IncBulkUpdateTestTrigger(nil)

		return c.JSON(testTriggers)
	}
}

// GetTestTriggerHandler is a handler for getting TestTrigger object
func (s *TestkubeAPI) GetTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test trigger %s", name)

		apiTestTrigger, err := s.TestTriggersClient.Get(c.Context(), s.getEnvironmentId(), name, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test trigger: %w", errPrefix, err))
		}

		c.Status(http.StatusOK)

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, []testkube.TestTrigger{*apiTestTrigger})
			return apiutils.SendLegacyCRDs(c, data, err)
		}

		return c.JSON(apiTestTrigger)
	}
}

// DeleteTestTriggerHandler is a handler for deleting TestTrigger by id
func (s *TestkubeAPI) DeleteTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete test trigger %s", name)

		err := s.TestTriggersClient.Delete(c.Context(), s.getEnvironmentId(), name, namespace)

		s.Metrics.IncDeleteTestTrigger(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test trigger: %w", errPrefix, err))
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// DeleteTestTriggersHandler is a handler for deleting all or selected TestTriggers
func (s *TestkubeAPI) DeleteTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete test triggers"

		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")
		if selector != "" {
			_, err := labels.Parse(selector)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: error validating selector: %w", errPrefix, err))
			}
		}
		_, err := s.TestTriggersClient.DeleteByLabels(c.Context(), s.getEnvironmentId(), selector, namespace)

		s.Metrics.IncBulkDeleteTestTrigger(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not bulk delete test triggers: %w", errPrefix, err))
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// ListTestTriggersHandler is a handler for listing all available TestTriggers
func (s *TestkubeAPI) ListTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test triggers"

		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")
		if selector != "" {
			_, err := labels.Parse(selector)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: error validating selector: %w", errPrefix, err))
			}
		}
		options := testtriggerclient.ListOptions{
			Selector: selector,
		}

		apiTestTriggers, err := s.TestTriggersClient.List(c.Context(), s.getEnvironmentId(), options, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test triggers: %w", errPrefix, err))
		}

		c.Status(http.StatusOK)

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, apiTestTriggers)
			return apiutils.SendLegacyCRDs(c, data, err)
		}

		return c.JSON(apiTestTriggers)
	}
}

// GetTestTriggerKeyMapHandler is a handler for listing supported TestTrigger field combinations
func (s *TestkubeAPI) GetTestTriggerKeyMapHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(triggerskeymapmapper.MapTestTriggerKeyMapToAPI(triggers.NewKeyMap()))
	}
}

// generateTestTriggerName function generates a trigger name from the TestTrigger spec
// function also takes care of name collisions, not exceeding k8s max object name (63 characters) and not ending with a hyphen '-'
func generateTestTriggerName(t *testtriggersv1.TestTrigger) string {
	name := fmt.Sprintf("trigger-%s-%s-%s-%s", t.Spec.Resource, t.Spec.Event, t.Spec.Action, t.Spec.Execution)
	if len(name) > testTriggerMaxNameLength {
		name = name[:testTriggerMaxNameLength-1]
	}
	name = strings.TrimSuffix(name, "-")
	name = fmt.Sprintf("%s-%s", name, utils.RandAlphanum(5))
	return name
}
