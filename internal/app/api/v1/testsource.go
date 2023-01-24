package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"k8s.io/apimachinery/pkg/api/errors"

	testsourcev1 "github.com/kubeshop/testkube-operator/apis/testsource/v1"
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
			if request.Data != "" {
				request.Data = fmt.Sprintf("%q", request.Data)
			}

			data, err := crd.GenerateYAML(crd.TemplateTestSource, []testkube.TestSourceUpsertRequest{request})
			return s.getCRDs(c, data, err)
		}

		testSource := testsourcesmapper.MapAPIToCRD(request)
		testSource.Namespace = s.Namespace
		var secrets map[string]string
		if request.Repository != nil {
			secrets = getTestSecretsData(request.Repository.Username, request.Repository.Token)
		}

		created, err := s.TestSourcesClient.Create(&testSource, testsources.Option{Secrets: secrets})
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) UpdateTestSourceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.TestSourceUpdateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		var name string
		if request.Name != nil {
			name = *request.Name
		}

		// we need to get resource first and load its metadata.ResourceVersion
		testSource, err := s.TestSourcesClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// map update test source but load spec only to not override metadata.ResourceVersion
		testSourceSpec := testsourcesmapper.MapUpdateToSpec(request, testSource)

		var option *testsources.Option
		if request.Repository != nil && (*request.Repository) != nil {
			username := (*request.Repository).Username
			token := (*request.Repository).Token
			if username != nil || token != nil {
				var uValue, tValue string
				if username != nil {
					uValue = *username
				}

				if token != nil {
					tValue = *token
				}

				option = &testsources.Option{Secrets: getTestSecretsData(uValue, tValue)}
			}
		}

		var updatedTestSource *testsourcev1.TestSource
		if option != nil {
			updatedTestSource, err = s.TestSourcesClient.Update(testSourceSpec, *option)
		} else {
			updatedTestSource, err = s.TestSourcesClient.Update(testSourceSpec)
		}

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(updatedTestSource)
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
			for i := range results {
				if results[i].Data != "" {
					results[i].Data = fmt.Sprintf("%q", results[i].Data)
				}
			}

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
			if result.Data != "" {
				result.Data = fmt.Sprintf("%q", result.Data)
			}

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

func (s TestkubeAPI) ProcessTestSourceBatchHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.TestSourceBatchRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		testSourceBatch := make(map[string]testkube.TestSourceUpsertRequest, len(request.Batch))
		for _, item := range request.Batch {
			if _, ok := testSourceBatch[item.Name]; ok {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("test source with duplicated id/name %s", item.Name))
			}

			testSourceBatch[item.Name] = item
		}

		list, err := s.TestSourcesClient.List("")
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		testSourceMap := make(map[string]testsourcev1.TestSource, len(list.Items))
		for _, item := range list.Items {
			testSourceMap[item.Name] = item
		}

		var result testkube.TestSourceBatchResult
		for name, item := range testSourceBatch {
			testSource := testsourcesmapper.MapAPIToCRD(item)
			var username, token string
			if item.Repository != nil {
				username = item.Repository.Username
				token = item.Repository.Token
			}

			if existed, ok := testSourceMap[name]; !ok {
				testSource.Namespace = s.Namespace

				created, err := s.TestSourcesClient.Create(&testSource, testsources.Option{Secrets: getTestSourceSecretsData(username, token)})
				if err != nil {
					return s.Error(c, http.StatusBadRequest, err)
				}

				result.Created = append(result.Created, created.Name)
			} else {
				existed.Spec = testSource.Spec
				existed.Labels = item.Labels

				updated, err := s.TestSourcesClient.Update(&existed, testsources.Option{Secrets: getTestSourceSecretsData(username, token)})
				if err != nil {
					return s.Error(c, http.StatusBadGateway, err)
				}

				result.Updated = append(result.Updated, updated.Name)
			}
		}

		for name := range testSourceMap {
			if _, ok := testSourceBatch[name]; !ok {
				err := s.TestSourcesClient.Delete(name)
				if err != nil {
					return s.Error(c, http.StatusBadRequest, err)
				}

				result.Deleted = append(result.Deleted, name)
			}
		}

		return c.JSON(result)
	}
}

func getTestSourceSecretsData(username, token string) map[string]string {
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
