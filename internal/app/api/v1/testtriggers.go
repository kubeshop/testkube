package v1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kubeshop/testkube/pkg/utils"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/gofiber/fiber/v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/keymap/triggers"
	triggerskeymapmapper "github.com/kubeshop/testkube/pkg/mapper/keymap/triggers"
	testtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/testtriggers"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
)

const testTriggerMaxNameLength = 57

// CreateTestTriggerHandler is a handler for creating test trigger objects
func (s *TestkubeAPI) CreateTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.TestTriggerUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		testTrigger := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(request)
		// default namespace if not defined in upsert request
		if testTrigger.Namespace == "" {
			testTrigger.Namespace = s.Namespace
		}
		// default trigger name if not defined in upsert request
		if testTrigger.Name == "" {
			testTrigger.Name = generateTestTriggerName(&testTrigger)
		}

		s.Log.Infow("creating test trigger", "testTrigger", testTrigger)

		created, err := s.TestKubeClientset.TestsV1().TestTriggers(s.Namespace).Create(c.UserContext(), &testTrigger, v1.CreateOptions{})

		s.Metrics.IncCreateTestTrigger(err)

		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusCreated)

		apiTestTrigger := testtriggersmapper.MapCRDToAPI(created)

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, []testkube.TestTrigger{apiTestTrigger})
			return s.getCRDs(c, data, err)
		}

		return c.JSON(apiTestTrigger)
	}
}

// UpdateTestTriggerHandler is a handler for updates an existing TestTrigger CRD based on TestTrigger content
func (s *TestkubeAPI) UpdateTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.TestTriggerUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		namespace := s.Namespace
		if request.Namespace != "" {
			namespace = request.Namespace
		}

		// we need to get resource first and load its metadata.ResourceVersion
		testTrigger, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Get(c.UserContext(), request.Name, v1.GetOptions{})
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		// map TestSuite but load spec only to not override metadata.ResourceVersion
		crdTestTrigger := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(request)
		testTrigger.Spec = crdTestTrigger.Spec
		testTrigger.Labels = request.Labels
		testTrigger, err = s.TestKubeClientset.TestsV1().TestTriggers(namespace).Update(c.UserContext(), testTrigger, v1.UpdateOptions{})

		s.Metrics.IncUpdateTestTrigger(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		apiTestTrigger := testtriggersmapper.MapCRDToAPI(testTrigger)

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, []testkube.TestTrigger{apiTestTrigger})
			return s.getCRDs(c, data, err)
		}

		return c.JSON(apiTestTrigger)
	}
}

// BulkUpdateTestTriggersHandler is a handler for bulk updates an existing TestTrigger CRDs based on array of TestTrigger content
func (s *TestkubeAPI) BulkUpdateTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request []testkube.TestTriggerUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		namespaces, err := s.Clientset.CoreV1().Namespaces().List(c.UserContext(), v1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "error fetching list of all namespaces")
		}
		for _, ns := range namespaces.Items {
			err = s.TestKubeClientset.
				TestsV1().
				TestTriggers(ns.Name).
				DeleteCollection(c.UserContext(), v1.DeleteOptions{}, v1.ListOptions{})

			if err != nil && !k8serrors.IsNotFound(err) {
				return s.Error(c, http.StatusBadGateway, errors.Wrap(err, "error cleaning triggers before reapply"))
			}
		}

		s.Metrics.IncBulkDeleteTestTrigger(nil)

		testTriggers := make([]testkube.TestTrigger, 0, len(request))

		namespace := s.Namespace
		for _, upsertRequest := range request {
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
				return errors.Wrap(err, "error reapplying triggers after clean")
			}

			testTriggers = append(testTriggers, testtriggersmapper.MapCRDToAPI(testTrigger))
		}

		s.Metrics.IncBulkUpdateTestTrigger(nil)

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, testTriggers)
			return s.getCRDs(c, data, err)
		}

		return c.JSON(testTriggers)
	}
}

// GetTestTriggerHandler is a handler for getting TestTrigger object
func (s *TestkubeAPI) GetTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		testTrigger, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Get(c.UserContext(), name, v1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
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
		err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Delete(c.UserContext(), name, v1.DeleteOptions{})

		s.Metrics.IncDeleteTestTrigger(err)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// DeleteTestTriggersHandler is a handler for deleting all or selected TestTriggers
func (s *TestkubeAPI) DeleteTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")
		if selector != "" {
			_, err := labels.Parse(selector)
			if err != nil {
				return errors.WithMessage(err, "error validating selector")
			}
		}
		listOpts := v1.ListOptions{LabelSelector: selector}
		err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).DeleteCollection(c.UserContext(), v1.DeleteOptions{}, listOpts)

		s.Metrics.IncBulkDeleteTestTrigger(err)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// ListTestTriggersHandler is a handler for listing all available TestTriggers
func (s *TestkubeAPI) ListTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")
		if selector != "" {
			_, err := labels.Parse(selector)
			if err != nil {
				return errors.WithMessage(err, "error validating selector")
			}
		}
		opts := v1.ListOptions{LabelSelector: selector}
		testTriggers, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).List(c.UserContext(), opts)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
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
