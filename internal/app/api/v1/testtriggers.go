package v1

import (
	"github.com/gofiber/fiber/v2"
	testtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/testtriggers"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"

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

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, []testkube.TestTriggerUpsertRequest{request})
			return s.getCRDs(c, data, err)
		}

		testTrigger := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(request)
		if testTrigger.Namespace == "" {
			testTrigger.Namespace = s.Namespace
		}

		s.Log.Infow("creating test trigger", "testTrigger", testTrigger)

		created, err := s.TestKubeClientset.TestsV1().TestTriggers(s.Namespace).Create(c.UserContext(), &testTrigger, v1.CreateOptions{})

		s.Metrics.IncCreateTestSuite(err)

		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

// UpdateTestTriggerHandler updates an existing TestTrigger CRD based on TestTrigger content
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

		return c.JSON(updated)
	}
}

// GetTestTriggerHandler for getting TestTrigger object
func (s TestkubeAPI) GetTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		crdTestTrigger, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Get(c.UserContext(), name, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		testTrigger := testtriggersmapper.MapCRDToAPI(crdTestTrigger)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, []testkube.TestTrigger{testTrigger})
			return s.getCRDs(c, data, err)
		}

		return c.JSON(testTrigger)
	}
}

// DeleteTestTriggerHandler for deleting TestTrigger by id
func (s TestkubeAPI) DeleteTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).Delete(c.UserContext(), name, v1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// DeleteTestTriggersHandler for deleting all or selected TestTriggers
func (s TestkubeAPI) DeleteTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var err error
		var testTriggerNames []string

		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")
		if selector == "" {
			err = s.TestKubeClientset.TestsV1().TestTriggers(namespace).DeleteCollection(c.UserContext(), v1.DeleteOptions{}, v1.ListOptions{})
		} else {
			listOpts := v1.ListOptions{LabelSelector: selector}
			testTriggerList, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).List(c.UserContext(), listOpts)
			if err != nil {
				if !errors.IsNotFound(err) {
					return s.Error(c, http.StatusBadGateway, err)
				}
			} else {
				for _, item := range testTriggerList.Items {
					testTriggerNames = append(testTriggerNames, item.Name)
				}
			}
			err = s.TestKubeClientset.TestsV1().TestTriggers(namespace).DeleteCollection(c.UserContext(), v1.DeleteOptions{}, listOpts)
		}

		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// ListTestTriggersHandler for listing all available TestTriggers
func (s *TestkubeAPI) ListTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		selector := c.Query("selector")
		namespace := c.Query("namespace", s.Namespace)
		opts := v1.ListOptions{LabelSelector: selector}
		crdTestTriggers, err := s.TestKubeClientset.TestsV1().TestTriggers(namespace).List(c.UserContext(), opts)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		testTriggers := testtriggersmapper.MapTestTriggerListKubeToAPI(crdTestTriggers)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestTrigger, testTriggers)
			return s.getCRDs(c, data, err)
		}

		return c.JSON(testTriggers)
	}
}
