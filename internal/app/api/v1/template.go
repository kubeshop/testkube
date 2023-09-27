package v1

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"

	templatev1 "github.com/kubeshop/testkube-operator/api/template/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	templatesmapper "github.com/kubeshop/testkube/pkg/mapper/templates"
)

func (s TestkubeAPI) CreateTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create template"
		var template templatev1.Template
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			templateSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(templateSpec), len(templateSpec))
			if err := decoder.Decode(&template); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}
		} else {
			var request testkube.TemplateCreateRequest
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}

			if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
				if request.Body != "" {
					request.Body = fmt.Sprintf("%q", request.Body)
				}

				data, err := crd.GenerateYAML(crd.TemplateTemplate, []testkube.TemplateCreateRequest{request})
				return s.getCRDs(c, data, err)
			}

			template = templatesmapper.MapAPIToCRD(request)
			template.Namespace = s.Namespace
		}

		created, err := s.TemplatesClient.Create(&template)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create template: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) UpdateTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to update template"
		var request testkube.TemplateUpdateRequest
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			var template templatev1.Template
			templateSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(templateSpec), len(templateSpec))
			if err := decoder.Decode(&template); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}

			request = templatesmapper.MapSpecToUpdate(&template)
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
		template, err := s.TemplatesClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client found no template: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get template: %w", errPrefix, err))
		}

		// map update template but load spec only to not override metadata.ResourceVersion
		templateSpec := templatesmapper.MapUpdateToSpec(request, template)

		updatedTemplate, err := s.TemplatesClient.Update(templateSpec)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update template: %w", errPrefix, err))
		}

		return c.JSON(updatedTemplate)
	}
}

func (s TestkubeAPI) ListTemplatesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list templates"

		list, err := s.TemplatesClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list templates: %w", errPrefix, err))
		}

		results := []testkube.Template{}
		for _, item := range list.Items {
			results = append(results, templatesmapper.MapCRDToAPI(item))

		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range results {
				if results[i].Body != "" {
					results[i].Body = fmt.Sprintf("%q", results[i].Body)
				}
			}

			data, err := crd.GenerateYAML(crd.TemplateTemplate, results)
			return s.getCRDs(c, data, err)
		}

		return c.JSON(results)
	}
}

func (s TestkubeAPI) GetTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		errPrefix := fmt.Sprintf("failed to get template %s", name)

		item, err := s.TemplatesClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: template not found: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get template: %w", errPrefix, err))
		}

		result := templatesmapper.MapCRDToAPI(*item)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			if result.Body != "" {
				result.Body = fmt.Sprintf("%q", result.Body)
			}

			data, err := crd.GenerateYAML(crd.TemplateTemplate, []testkube.Template{result})
			return s.getCRDs(c, data, err)
		}

		return c.JSON(result)
	}
}

func (s TestkubeAPI) DeleteTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		errPrefix := fmt.Sprintf("failed to delete template %s", name)

		err := s.TemplatesClient.Delete(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: template not found: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete template: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func (s TestkubeAPI) DeleteTemplatesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete templates"

		err := s.TemplatesClient.DeleteByLabels(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete templates: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}
