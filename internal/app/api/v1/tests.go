package v1

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/secret"

	"github.com/kubeshop/testkube/pkg/jobs"
	"k8s.io/apimachinery/pkg/api/errors"
)

// GetTestHandler is method for getting an existing tests
func (s TestkubeAPI) GetTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		crTest, err := s.TestsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		tests := testsmapper.MapTestCRToAPI(*crTest)
		return c.JSON(tests)
	}
}

// ListTestsHandler is a method for getting list of all available tests
func (s TestkubeAPI) ListTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")

		rawTags := c.Query("tags")
		var tags []string
		if rawTags != "" {
			tags = strings.Split(rawTags, ",")
		}

		// TODO filters looks messy need to introduce some common Filter object for Kubernetes query for List like objects
		crTests, err := s.TestsClient.List(namespace, tags)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		search := c.Query("textSearch")
		if search != "" {
			// filter items array
			for i := len(crTests.Items) - 1; i >= 0; i-- {
				if !strings.Contains(crTests.Items[i].Name, search) {
					crTests.Items = append(crTests.Items[:i], crTests.Items[i+1:]...)
				}
			}
		}

		testType := c.Query("type")
		if testType != "" {
			// filter items array
			for i := len(crTests.Items) - 1; i >= 0; i-- {
				if !strings.Contains(crTests.Items[i].Spec.Type_, testType) {
					crTests.Items = append(crTests.Items[:i], crTests.Items[i+1:]...)
				}
			}
		}

		tests := testsmapper.MapTestListKubeToAPI(*crTests)

		return c.JSON(tests)
	}
}

// CreateTestHandler creates new test CR based on test content
func (s TestkubeAPI) CreateTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.TestUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		s.Log.Infow("creating test", "request", request)

		testSpec := testsmapper.MapToSpec(request)
		test, err := s.TestsClient.Create(testSpec)

		s.Metrics.IncCreateTest(test.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		stringData := GetSecretsStringData(request.Content)
		if err = s.SecretClient.Create(secret.GetMetadataName(request.Name), request.Namespace, stringData); err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(test)
	}
}

// UpdateTestHandler updates an existing test CR based on test content
func (s TestkubeAPI) UpdateTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.TestUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		s.Log.Infow("updating test", "request", request)

		// we need to get resource first and load its metadata.ResourceVersion
		test, err := s.TestsClient.Get(request.Namespace, request.Name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		// map test but load spec only to not override metadata.ResourceVersion
		testSpec := testsmapper.MapToSpec(request)
		test.Spec = testSpec.Spec
		test, err = s.TestsClient.Update(test)

		s.Metrics.IncUpdateTest(test.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		// update secrets for scipt
		stringData := GetSecretsStringData(request.Content)
		if err = s.SecretClient.Apply(secret.GetMetadataName(request.Name), request.Namespace, stringData); err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(test)
	}
}

// DeleteTestHandler is a method for deleting a test with id
func (s TestkubeAPI) DeleteTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		err := s.TestsClient.Delete(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// delete secrets for test
		if err = s.SecretClient.Delete(secret.GetMetadataName(name), namespace); err != nil {
			if errors.IsNotFound(err) {
				return c.SendStatus(fiber.StatusNoContent)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// DeleteTestsHandler for deleting all tests
func (s TestkubeAPI) DeleteTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")
		err := s.TestsClient.DeleteAll(namespace)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// delete all secrets for tests
		if err = s.SecretClient.DeleteAll(namespace); err != nil {
			if errors.IsNotFound(err) {
				return c.SendStatus(fiber.StatusNoContent)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

func GetSecretsStringData(content *testkube.TestContent) map[string]string {
	// create secrets for test
	stringData := map[string]string{jobs.GitUsernameSecretName: "", jobs.GitTokenSecretName: ""}
	if content != nil && content.Repository != nil {
		stringData[jobs.GitUsernameSecretName] = content.Repository.Username
		stringData[jobs.GitTokenSecretName] = content.Repository.Token
	}

	return stringData
}
