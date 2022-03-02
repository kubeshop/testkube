package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s TestkubeAPI) CreateExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.ExecutorCreateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		executor := mapExecutorCreateRequestToExecutorCRD(request)
		if executor.Spec.JobTemplate == "" {
			executor.Spec.JobTemplate = s.jobTemplate
		}

		created, err := s.ExecutorsClient.Create(&executor)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(201)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) ListExecutorsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ns := c.Query("namespace", "testkube")
		list, err := s.ExecutorsClient.List(ns)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
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
		ns := c.Query("namespace", "testkube")

		item, err := s.ExecutorsClient.Get(ns, name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}
		result := mapExecutorCRDToExecutorDetails(*item)

		return c.JSON(result)
	}
}

func (s TestkubeAPI) DeleteExecutorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		ns := c.Query("namespace", "testkube")

		err := s.ExecutorsClient.Delete(name, ns)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Context().SetStatusCode(204)
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
		},
	}
}

func mapExecutorCreateRequestToExecutorCRD(request testkube.ExecutorCreateRequest) executorv1.Executor {
	return executorv1.Executor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
		},
		Spec: executorv1.ExecutorSpec{
			ExecutorType: request.ExecutorType,
			Types:        request.Types,
			URI:          request.Uri,
			Image:        request.Image,
			JobTemplate:  request.JobTemplate,
		},
	}
}
