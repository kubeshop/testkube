package v1

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"

	testsourcev1 "github.com/kubeshop/testkube-operator/api/testsource/v1"
	"github.com/kubeshop/testkube-operator/pkg/client/testsources/v1"
	"github.com/kubeshop/testkube-operator/pkg/secret"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/executor/client"
	testsourcesmapper "github.com/kubeshop/testkube/pkg/mapper/testsources"
)

func (s TestkubeAPI) CreateTestSourceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create test source"
		var testSource testsourcev1.TestSource
		var secrets map[string]string
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			testSourceSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(testSourceSpec), len(testSourceSpec))
			if err := decoder.Decode(&testSource); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}
		} else {
			var request testkube.TestSourceUpsertRequest
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %s", errPrefix, err))
			}

			if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
				if request.Data != "" {
					request.Data = fmt.Sprintf("%q", request.Data)
				}

				data, err := crd.GenerateYAML(crd.TemplateTestSource, []testkube.TestSourceUpsertRequest{request})
				return s.getCRDs(c, data, err)
			}

			testSource = testsourcesmapper.MapAPIToCRD(request)
			testSource.Namespace = s.Namespace
			if request.Repository != nil && !s.disableSecretCreation {
				secrets = createTestSecretsData(request.Repository.Username, request.Repository.Token)
			}
		}

		created, err := s.TestSourcesClient.Create(&testSource, testsources.Option{Secrets: secrets})
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create test source: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) UpdateTestSourceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to update test source"
		var request testkube.TestSourceUpdateRequest
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			var testSource testsourcev1.TestSource
			testSourceSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(testSourceSpec), len(testSourceSpec))
			if err := decoder.Decode(&testSource); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}

			request = testsourcesmapper.MapSpecToUpdate(&testSource)
		} else {
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json jrequest: %s", errPrefix, err))
			}
		}

		var name string
		if request.Name != nil {
			name = *request.Name
		}
		errPrefix = errPrefix + " " + name
		// we need to get resource first and load its metadata.ResourceVersion
		testSource, err := s.TestSourcesClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test source not found: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// map update test source but load spec only to not override metadata.ResourceVersion
		testSourceSpec := testsourcesmapper.MapUpdateToSpec(request, testSource)

		var option *testsources.Option
		if request.Repository != nil && (*request.Repository) != nil {
			username := (*request.Repository).Username
			token := (*request.Repository).Token
			if (username != nil || token != nil) && !s.disableSecretCreation {
				data, err := s.SecretClient.Get(secret.GetMetadataName(name, client.SecretSource))
				if err != nil && !errors.IsNotFound(err) {
					return s.Error(c, http.StatusBadGateway, err)
				}

				option = &testsources.Option{Secrets: updateTestSecretsData(data, username, token)}
			}
		}

		var updatedTestSource *testsourcev1.TestSource
		if option != nil {
			updatedTestSource, err = s.TestSourcesClient.Update(testSourceSpec, *option)
		} else {
			updatedTestSource, err = s.TestSourcesClient.Update(testSourceSpec)
		}

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client coult not update test source: %w", errPrefix, err))
		}

		return c.JSON(updatedTestSource)
	}
}

func (s TestkubeAPI) ListTestSourcesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test sources"

		list, err := s.TestSourcesClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test sources: %s", errPrefix, err))
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
		errPrefix := "failed to get test source" + name

		item, err := s.TestSourcesClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client could not find test source: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test source: %w", errPrefix, err))
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
		errPrefix := "failed to delete test source" + name

		err := s.TestSourcesClient.Delete(name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test source: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func (s TestkubeAPI) DeleteTestSourcesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete test sources"
		err := s.TestSourcesClient.DeleteByLabels(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test sources: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func (s TestkubeAPI) ProcessTestSourceBatchHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to batch process test sources"

		var request testkube.TestSourceBatchRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse request: %s", errPrefix, err))
		}

		testSourceBatch := make(map[string]testkube.TestSourceUpsertRequest, len(request.Batch))
		for _, item := range request.Batch {
			if _, ok := testSourceBatch[item.Name]; ok {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: test source with duplicated id/name %s", errPrefix, item.Name))
			}

			testSourceBatch[item.Name] = item
		}

		list, err := s.TestSourcesClient.List("")
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test sources: %w", errPrefix, err))
		}

		testSourceMap := make(map[string]testsourcev1.TestSource, len(list.Items))
		for _, item := range list.Items {
			testSourceMap[item.Name] = item
		}

		var result testkube.TestSourceBatchResult
		for name, item := range testSourceBatch {
			testSource := testsourcesmapper.MapAPIToCRD(item)
			var username, token string
			if item.Repository != nil && !s.disableSecretCreation {
				username = item.Repository.Username
				token = item.Repository.Token
			}

			if existed, ok := testSourceMap[name]; !ok {
				testSource.Namespace = s.Namespace

				created, err := s.TestSourcesClient.Create(&testSource, testsources.Option{Secrets: getTestSourceSecretsData(username, token)})
				if err != nil {
					return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create test source %s: %w", errPrefix, testSource.Name, err))
				}

				result.Created = append(result.Created, created.Name)
			} else {
				existed.Spec = testSource.Spec
				existed.Labels = item.Labels

				updated, err := s.TestSourcesClient.Update(&existed, testsources.Option{Secrets: getTestSourceSecretsData(username, token)})
				if err != nil {
					return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update test source %s: %w", errPrefix, testSource.Name, err))
				}

				result.Updated = append(result.Updated, updated.Name)
			}
		}

		for name := range testSourceMap {
			if _, ok := testSourceBatch[name]; !ok {
				err := s.TestSourcesClient.Delete(name)
				if err != nil {
					return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test source %s: %w", errPrefix, name, err))
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
