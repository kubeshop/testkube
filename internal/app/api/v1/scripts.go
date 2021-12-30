package v1

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	scriptsv1 "github.com/kubeshop/testkube-operator/apis/script/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	scriptsMapper "github.com/kubeshop/testkube/pkg/mapper/scripts"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/pkg/api/datefilter"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
)

// ListScripts for getting list of all available scripts
func (s TestKubeAPI) GetScriptHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		crScript, err := s.ScriptsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptCRToAPI(*crScript)
		return c.JSON(scripts)
	}
}

// ListScriptsHandler for getting list of all available scripts
func (s TestKubeAPI) ListScriptsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")

		raw_tags := c.Query("tags")
		var tags []string
		if raw_tags != "" {
			tags = strings.Split(raw_tags, ",")
		}

		crScripts, err := s.ScriptsClient.List(namespace, tags)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptListKubeToAPI(*crScripts)

		return c.JSON(scripts)
	}
}

// CreateScriptHandler creates new script CR based on script content
func (s TestKubeAPI) CreateScriptHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.ScriptUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		s.Log.Infow("creating script", "request", request)

		var repository *scriptsv1.Repository

		if request.Repository != nil {
			repository = &scriptsv1.Repository{
				Type_:  "git",
				Uri:    request.Repository.Uri,
				Branch: request.Repository.Branch,
				Path:   request.Repository.Path,
			}
		}

		script, err := s.ScriptsClient.Create(&scriptsv1.Script{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
			},
			Spec: scriptsv1.ScriptSpec{
				Type_:      request.Type_,
				InputType:  request.InputType,
				Content:    request.Content,
				Repository: repository,
				Tags:       request.Tags,
			},
		})

		s.Metrics.IncCreateScript(script.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(script)
	}
}

// UpdateScriptHandler creates new script CR based on script content
func (s TestKubeAPI) UpdateScriptHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.ScriptUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		s.Log.Infow("updating script", "request", request)

		var repository *scriptsv1.Repository

		if request.Repository != nil {
			repository = &scriptsv1.Repository{
				Type_:  "git",
				Uri:    request.Repository.Uri,
				Branch: request.Repository.Branch,
				Path:   request.Repository.Path,
			}
		}

		// we need to get resouece first and load its metadata.ResourceVersion
		script, err := s.ScriptsClient.Get(request.Namespace, request.Name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		script.Spec = scriptsv1.ScriptSpec{
			Type_:      request.Type_,
			InputType:  request.InputType,
			Content:    request.Content,
			Repository: repository,
			Tags:       request.Tags,
		}

		script, err = s.ScriptsClient.Update(script)

		s.Metrics.IncUpdateScript(script.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(script)
	}
}

// DeleteScriptHandler for deleting a script with id
func (s TestKubeAPI) DeleteScriptHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		err := s.ScriptsClient.Delete(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// DeleteScriptsHandler for deleting all scripts
func (s TestKubeAPI) DeleteScriptsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")
		err := s.ScriptsClient.DeleteAll(namespace)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

func getFilterFromRequest(c *fiber.Ctx) result.Filter {

	filter := result.NewExecutionsFilter()
	scriptName := c.Params("id", "-")
	if scriptName != "-" {
		filter = filter.WithScriptName(scriptName)
	}

	textSearch := c.Query("textSearch", "")
	if textSearch != "" {
		filter = filter.WithTextSearch(textSearch)
	}

	page, err := strconv.Atoi(c.Query("page", "-"))
	if err == nil {
		filter = filter.WithPage(page)
	}

	pageSize, err := strconv.Atoi(c.Query("pageSize", "-"))
	if err == nil && pageSize != 0 {
		filter = filter.WithPageSize(pageSize)
	}

	status := c.Query("status", "-")
	if status != "-" {
		filter = filter.WithStatus(testkube.ExecutionStatus(status))
	}

	dFilter := datefilter.NewDateFilter(c.Query("startDate", ""), c.Query("endDate", ""))
	if dFilter.IsStartValid {
		filter = filter.WithStartDate(dFilter.Start)
	}

	if dFilter.IsEndValid {
		filter = filter.WithEndDate(dFilter.End)
	}

	raw_tags := c.Query("tags")
	if raw_tags != "" {
		filter = filter.WithTags(strings.Split(raw_tags, ","))
	}

	return filter
}
