package v1

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/yaml"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/keymap/triggers"
	triggerskeymapmapper "github.com/kubeshop/testkube/pkg/mapper/keymap/triggers"
	testtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/testtriggers"
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
				return s.getCRDs(c, data, err)
			}
		}

		errPrefix = errPrefix + " " + testTrigger.Name

		s.Log.Infow("creating test trigger", "testTrigger", testTrigger)

		created, err := s.TestKubeClientset.TestsV1().TestTriggers(s.Namespace).Create(c.UserContext(), &testTrigger, v1.CreateOptions{})

		s.Metrics.IncCreateTestTrigger(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create test trigger: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(testtriggersmapper.MapCRDToAPI(created))
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

		// we need to get resource first and load its metadata.ResourceVersion
		testTrigger, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Get(c.UserContext(), request.Name, v1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client could not find test trigger: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test trigger: %w", errPrefix, err))
		}

		// map TestSuite but load spec only to not override metadata.ResourceVersion
		crdTestTrigger := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(request)
		testTrigger.Spec = crdTestTrigger.Spec
		testTrigger.Labels = request.Labels
		testTrigger, err = s.TestKubeClientset.TestsV1().TestTriggers(namespace).Update(c.UserContext(), testTrigger, v1.UpdateOptions{})

		s.Metrics.IncUpdateTestTrigger(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update test trigger: %w", errPrefix, err))
		}

		return c.JSON(testtriggersmapper.MapCRDToAPI(testTrigger))
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
			err = s.TestKubeClientset.
				TestsV1().
				TestTriggers(namespace).
				DeleteCollection(c.UserContext(), v1.DeleteOptions{}, v1.ListOptions{})
			if err != nil && !k8serrors.IsNotFound(err) {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: error cleaning triggers before reapply", errPrefix))
			}
		}

		s.Metrics.IncBulkDeleteTestTrigger(nil)

		testTriggers := make([]testkube.TestTrigger, 0, len(request))

		for _, upsertRequest := range request {
			namespace := s.Namespace
			if upsertRequest.Namespace != "" {
				namespace = upsertRequest.Namespace
			}
			var testTrigger *testtriggersv1.TestTrigger
			crdTestTrigger := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(upsertRequest)
			// default trigger name if not defined in upsert request
			if crdTestTrigger.Name == "" {
				crdTestTrigger.Name = generateTestTriggerName(&crdTestTrigger)
			}
			testTrigger, err = s.TestKubeClientset.
				TestsV1().
				TestTriggers(namespace).
				Create(c.UserContext(), &crdTestTrigger, v1.CreateOptions{})

			s.Metrics.IncCreateTestTrigger(err)

			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: error reapplying triggers after clean", errPrefix))
			}

			testTriggers = append(testTriggers, testtriggersmapper.MapCRDToAPI(testTrigger))
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

		testTrigger, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Get(c.UserContext(), name, v1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: client could not find test trigger: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test trigger: %w", errPrefix, err))
		}

		c.Status(http.StatusOK)

		apiTestTrigger := testtriggersmapper.MapCRDToAPI(testTrigger)

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, []testkube.TestTrigger{apiTestTrigger})
			return s.getCRDs(c, data, err)
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

		err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Delete(c.UserContext(), name, v1.DeleteOptions{})

		s.Metrics.IncDeleteTestTrigger(err)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: client could not find test trigger: %w", errPrefix, err))
			}

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
		listOpts := v1.ListOptions{LabelSelector: selector}
		err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).DeleteCollection(c.UserContext(), v1.DeleteOptions{}, listOpts)

		s.Metrics.IncBulkDeleteTestTrigger(err)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: client could not find test trigger: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not bulk delete test triggers: %w", errPrefix, err))
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// ListTestTriggersHandler is a handler for listing all available TestTriggers
func (s *TestkubeAPI) ListTestTriggersHandler() fiber.Handler {
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
		opts := v1.ListOptions{LabelSelector: selector}
		testTriggers, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).List(c.UserContext(), opts)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test triggers: %w", errPrefix, err))
		}

		c.Status(http.StatusOK)

		apiTestTriggers := testtriggersmapper.MapTestTriggerListKubeToAPI(testTriggers)

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, apiTestTriggers)
			return s.getCRDs(c, data, err)
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
