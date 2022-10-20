package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
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
			data, err := crd.GenerateYAML(crd.TemplateExecutor, []testkube.ExecutorUpsertRequest{request})
			return s.getCRDs(c, data, err)
		}

		executor := executorsmapper.MapAPIToCRD(request)
		if executor.Spec.ExecutorType != "container" && executor.Spec.JobTemplate == "" {
			executor.Spec.JobTemplate = s.jobTemplates.Job
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
		var request testkube.ExecutorUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		// we need to get resource first and load its metadata.ResourceVersion
		executor, err := s.ExecutorsClient.Get(request.Name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		executorSpec := executorsmapper.MapAPIToCRD(request)
		executor.Spec = executorSpec.Spec
		executor.Labels = request.Labels

		executor, err = s.ExecutorsClient.Update(executor)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(executor)
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
				if item.Spec.JobTemplate != "" {
					item.Spec.JobTemplate = fmt.Sprintf("%q", item.Spec.JobTemplate)
				}

				results = append(results, executorsmapper.MapCRDToAPI(item))
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
			if item.Spec.JobTemplate != "" {
				item.Spec.JobTemplate = fmt.Sprintf("%q", item.Spec.JobTemplate)
			}

			data, err := crd.GenerateYAML(crd.TemplateExecutor, []testkube.ExecutorUpsertRequest{executorsmapper.MapCRDToAPI(*item)})
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
