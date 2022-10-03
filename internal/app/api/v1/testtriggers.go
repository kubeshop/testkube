package v1

import (
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/keymap/triggers"
	triggerskeymapmapper "github.com/kubeshop/testkube/pkg/mapper/keymap/triggers"
	testtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/testtriggers"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
)

// CreateTestTriggerHandler is a handler for creating test trigger objects
func (s *TestkubeAPI) CreateTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.TestTriggerUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		testTrigger := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(request)
		if testTrigger.Namespace == "" {
			testTrigger.Namespace = s.Namespace
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

		namespace := c.Query("namespace", s.Namespace)

		// we need to get resource first and load its metadata.ResourceVersion
		testTrigger, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Get(c.UserContext(), request.Name, v1.GetOptions{})
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		// map TestSuite but load spec only to not override metadata.ResourceVersion
		testTriggerSpec := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(request)
		testTrigger.Spec = testTriggerSpec.Spec
		testTrigger.Labels = request.Labels
		updated, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Update(c.UserContext(), testTrigger, v1.UpdateOptions{})

		s.Metrics.IncUpdateTestTrigger(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		c.Status(http.StatusOK)

		apiTestTrigger := testtriggersmapper.MapCRDToAPI(updated)

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, []testkube.TestTrigger{apiTestTrigger})
			return s.getCRDs(c, data, err)
		}

		return c.JSON(apiTestTrigger)
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
		if err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).DeleteCollection(c.UserContext(), v1.DeleteOptions{}, listOpts); err != nil {
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
