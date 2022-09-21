package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube-operator/client/testsources/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/executor/client"
	testsourcesmapper "github.com/kubeshop/testkube/pkg/mapper/testsources"
)

func (s TestkubeAPI) CreateTestSourceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.TestSourceUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestSource, []testkube.TestSourceUpsertRequest{request})
			return s.getCRDs(c, data, err)
		}

		testSource := testsourcesmapper.MapAPIToCRD(request)
		testSource.Namespace = s.Namespace

		created, err := s.TestSourcesClient.Create(&testSource, testsources.Option{Secrets: getTestSourceSecretsData(&request)})
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) UpdateTestSourceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.TestSourceUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		// we need to get resource first and load its metadata.ResourceVersion
		testSource, err := s.TestSourcesClient.Get(request.Name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		testSourceSpec := testsourcesmapper.MapAPIToCRD(request)
		testSource.Spec = testSourceSpec.Spec
		testSource.Labels = request.Labels

		testSource, err = s.TestSourcesClient.Update(testSource, testsources.Option{Secrets: getTestSourceSecretsData(&request)})
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(testSource)
	}
}

func (s TestkubeAPI) ListTestSourcesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		list, err := s.TestSourcesClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		results := []testkube.TestSource{}
		for _, item := range list.Items {
			results = append(results, testsourcesmapper.MapCRDToAPI(item))

		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestSource, results)
			return s.getCRDs(c, data, err)
		}

		return c.JSON(results)
	}
}

func (s TestkubeAPI) GetTestSourceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")

		item, err := s.TestSourcesClient.Get(name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		result := testsourcesmapper.MapCRDToAPI(*item)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateTestSource, []testkube.TestSource{result})
			return s.getCRDs(c, data, err)
		}

		return c.JSON(result)
	}
}

func (s TestkubeAPI) DeleteTestSourceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")

		err := s.TestSourcesClient.Delete(name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func (s TestkubeAPI) DeleteTestSourcesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := s.TestSourcesClient.DeleteByLabels(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func getTestSourceSecretsData(testSource *testkube.TestSourceUpsertRequest) map[string]string {
	// create secrets for test
	username := ""
	token := ""
	if testSource.Repository != nil {
		username = testSource.Repository.Username
		token = testSource.Repository.Token
	}

	if username == "" && token == "" {
		return nil
	}

	data := make(map[string]string, 0)
	if username != "" {
		data[client.GitUsernameSecretName] = username
	}

	if token != "" {
		data[client.GitTokenSecretName] = token
	}

	return data
}
