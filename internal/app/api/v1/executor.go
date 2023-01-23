package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	executorsmapper "github.com/kubeshop/testkube/pkg/mapper/executors"
)

func (s TestkubeAPI) CreateExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.ExecutorUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			request.QuoteExecutorTextFields()
			data, err := crd.GenerateYAML(crd.TemplateExecutor, []testkube.ExecutorUpsertRequest{request})
			return s.getCRDs(c, data, err)
		}

		executor := executorsmapper.MapAPIToCRD(request)
		if executor.Spec.ExecutorType != "container" && executor.Spec.JobTemplate == "" {
			executor.Spec.JobTemplate = s.templates.Job
		}
		executor.Namespace = s.Namespace

		created, err := s.ExecutorsClient.Create(&executor)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) UpdateExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.ExecutorUpdateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		var name string
		if request.Name != nil {
			name = *request.Name
		}

		// we need to get resource first and load its metadata.ResourceVersion
		executor, err := s.ExecutorsClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// map update executor but load spec only to not override metadata.ResourceVersion
		executorSpec := executorsmapper.MapUpdateToSpec(request, executor)

		updatedExecutor, err := s.ExecutorsClient.Update(executorSpec)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(updatedExecutor)
	}
}

func (s TestkubeAPI) ListExecutorsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		list, err := s.ExecutorsClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
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
		item, err := s.ExecutorsClient.Get(name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
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

		err := s.ExecutorsClient.Delete(name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func (s TestkubeAPI) DeleteExecutorsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := s.ExecutorsClient.DeleteByLabels(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}
