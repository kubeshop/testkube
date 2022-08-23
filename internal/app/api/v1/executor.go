package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	executorsmapper "github.com/kubeshop/testkube/pkg/mapper/executors"
)

func (s TestkubeAPI) CreateExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.ExecutorCreateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			data, err := crd.GenerateYAML(crd.TemplateExecutor, []testkube.ExecutorCreateRequest{request})
			return s.getCRDs(c, data, err)
		}

		executor := executorsmapper.MapAPIToCRD(request)
		if executor.Spec.JobTemplate == "" {
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

func (s TestkubeAPI) ListExecutorsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		list, err := s.ExecutorsClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			results := []testkube.ExecutorCreateRequest{}
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
			results = append(results, mapExecutorCRDToExecutorDetails(item))

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

			data, err := crd.GenerateYAML(crd.TemplateExecutor, []testkube.ExecutorCreateRequest{executorsmapper.MapCRDToAPI(*item)})
			return s.getCRDs(c, data, err)
		}

		result := mapExecutorCRDToExecutorDetails(*item)
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

func mapExecutorCRDToExecutorDetails(item executorv1.Executor) testkube.ExecutorDetails {
	return testkube.ExecutorDetails{
		Name: item.Name,
		Executor: &testkube.Executor{
			ExecutorType: item.Spec.ExecutorType,
			Image:        item.Spec.Image,
			Types:        item.Spec.Types,
			Uri:          item.Spec.URI,
			JobTemplate:  item.Spec.JobTemplate,
			Labels:       item.Labels,
			Features:     mapFeatures(item.Spec.Features),
		},
	}
}

func mapFeatures(features []executorv1.Feature) (out []string) {

	for _, feature := range features {
		out = append(out, string(feature))
	}

	return
}
