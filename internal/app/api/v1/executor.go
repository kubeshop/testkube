package v1

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	executorsmapper "github.com/kubeshop/testkube/pkg/mapper/executors"
)

func (s TestkubeAPI) CreateExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create executor"
		var executor executorv1.Executor
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			executorSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(executorSpec), len(executorSpec))
			if err := decoder.Decode(&executor); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}
		} else {
			var request testkube.ExecutorUpsertRequest
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}

			if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
				request.QuoteExecutorTextFields()
				data, err := crd.GenerateYAML(crd.TemplateExecutor, []testkube.ExecutorUpsertRequest{request})
				return s.getCRDs(c, data, err)
			}

			executor = executorsmapper.MapAPIToCRD(request)
			executor.Namespace = s.Namespace
		}

		created, err := s.ExecutorsClient.Create(&executor)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create executor: %w", errPrefix, err))
		}

		s.Events.Notify(testkube.NewEvent(
			testkube.EventCreated,
			testkube.EventResourceExecutor,
			created.Name,
		))

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) UpdateExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to update executor"
		var request testkube.ExecutorUpdateRequest
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			var executor executorv1.Executor
			executorSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(executorSpec), len(executorSpec))
			if err := decoder.Decode(&executor); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}

			request = executorsmapper.MapSpecToUpdate(&executor)
		} else {
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}
		}

		var name string
		if request.Name != nil {
			name = *request.Name
		}
		errPrefix = errPrefix + " " + name
		// we need to get resource first and load its metadata.ResourceVersion
		executor, err := s.ExecutorsClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client found no executor: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get executor: %w", errPrefix, err))
		}

		// map update executor but load spec only to not override metadata.ResourceVersion
		executorSpec := executorsmapper.MapUpdateToSpec(request, executor)

		updatedExecutor, err := s.ExecutorsClient.Update(executorSpec)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update executor: %w", errPrefix, err))
		}

		s.Events.Notify(testkube.NewEvent(
			testkube.EventUpdated,
			testkube.EventResourceExecutor,
			updatedExecutor.Name,
		))

		return c.JSON(updatedExecutor)
	}
}

func (s TestkubeAPI) ListExecutorsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list executors"
		list, err := s.ExecutorsClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list executors: %w", errPrefix, err))
		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			results := []testkube.ExecutorUpsertRequest{}
			for _, item := range list.Items {
				result := executorsmapper.MapCRDToAPI(item)
				result.QuoteExecutorTextFields()
				results = append(results, result)
			}

			data, err := crd.GenerateYAML(crd.TemplateExecutor, results)
			return s.getCRDs(c, data, err)
		}

		results := []testkube.ExecutorDetails{}
		for _, item := range list.Items {
			results = append(results, executorsmapper.MapExecutorCRDToExecutorDetails(item))

		}
		return c.JSON(results)
	}
}

func (s TestkubeAPI) GetExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		errPrefix := fmt.Sprintf("failed to get executor %s", name)

		item, err := s.ExecutorsClient.Get(name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get executor: %w", errPrefix, err))
		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			result := executorsmapper.MapCRDToAPI(*item)
			result.QuoteExecutorTextFields()
			data, err := crd.GenerateYAML(crd.TemplateExecutor, []testkube.ExecutorUpsertRequest{result})
			return s.getCRDs(c, data, err)
		}

		result := executorsmapper.MapExecutorCRDToExecutorDetails(*item)
		return c.JSON(result)
	}
}

func (s TestkubeAPI) DeleteExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		errPrefix := fmt.Sprintf("failed to delete executor %s", name)

		err := s.ExecutorsClient.Delete(name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete executor: %w", errPrefix, err))
		}

		s.Events.Notify(testkube.NewEvent(
			testkube.EventDeleted,
			testkube.EventResourceExecutor,
			name,
		))

		c.Status(http.StatusNoContent)

		return nil
	}
}

func (s TestkubeAPI) DeleteExecutorsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete executors"
		err := s.ExecutorsClient.DeleteByLabels(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete executors: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func (s TestkubeAPI) GetExecutorByTestTypeHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to get executor by test type"

		testType := c.Query("testType", "")
		if testType == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not fine test type", errPrefix))
		}

		item, err := s.ExecutorsClient.GetByType(testType)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get executor: %w", errPrefix, err))
		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			result := executorsmapper.MapCRDToAPI(*item)
			result.QuoteExecutorTextFields()
			data, err := crd.GenerateYAML(crd.TemplateExecutor, []testkube.ExecutorUpsertRequest{result})
			return s.getCRDs(c, data, err)
		}

		result := executorsmapper.MapExecutorCRDToExecutorDetails(*item)
		return c.JSON(result)
	}
}
