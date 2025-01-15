package v1

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	webhooktemplatesmapper "github.com/kubeshop/testkube/pkg/mapper/webhooktemplates"
)

func (s *TestkubeAPI) CreateWebhookTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create webhook template"
		var webhookTemplate executorv1.WebhookTemplate
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			webhookTemplateSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(webhookTemplateSpec), len(webhookTemplateSpec))
			if err := decoder.Decode(&webhookTemplate); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}
		} else {
			var request testkube.WebhookTemplateCreateRequest
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}

			if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
				request.QuoteTextFields()
				data, err := crd.GenerateYAML(crd.TemplateWebhookTemplate, []testkube.WebhookTemplateCreateRequest{request})
				return apiutils.SendLegacyCRDs(c, data, err)
			}

			webhookTemplate = webhooktemplatesmapper.MapAPIToCRD(request)
			webhookTemplate.Namespace = s.Namespace
		}

		created, err := s.WebhookTemplatesClient.Create(&webhookTemplate)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create webhook template: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

func (s *TestkubeAPI) UpdateWebhookTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to update webhook template"
		var request testkube.WebhookTemplateUpdateRequest
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			var webhookTemplate executorv1.WebhookTemplate
			webhookTemplateSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(webhookTemplateSpec), len(webhookTemplateSpec))
			if err := decoder.Decode(&webhookTemplate); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}

			request = webhooktemplatesmapper.MapSpecToUpdate(&webhookTemplate)
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
		webhookTemplate, err := s.WebhookTemplatesClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client found no webhook template: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get webhook template: %w", errPrefix, err))
		}

		// map update webhook template but load spec only to not override metadata.ResourceVersion
		webhookTemplateSpec := webhooktemplatesmapper.MapUpdateToSpec(request, webhookTemplate)

		updatedWebhookTemplate, err := s.WebhookTemplatesClient.Update(webhookTemplateSpec)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update webhook template: %w", errPrefix, err))
		}

		return c.JSON(updatedWebhookTemplate)
	}
}

func (s *TestkubeAPI) ListWebhookTemplatesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list webhook templates"

		list, err := s.WebhookTemplatesClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list webhook templates: %w", errPrefix, err))
		}

		results := testkube.WebhookTemplates{}
		for _, item := range list.Items {
			results = append(results, webhooktemplatesmapper.MapCRDToAPI(item))

		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range results {
				results[i].QuoteTextFields()
			}

			data, err := crd.GenerateYAML(crd.TemplateWebhookTemplate, results)
			return apiutils.SendLegacyCRDs(c, data, err)
		}

		return c.JSON(results)
	}
}

func (s *TestkubeAPI) GetWebhookTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		errPrefix := fmt.Sprintf("failed to get webhook template %s", name)

		item, err := s.WebhookTemplatesClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: webhook template not found: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get webhook template: %w", errPrefix, err))
		}

		result := webhooktemplatesmapper.MapCRDToAPI(*item)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			result.QuoteTextFields()
			data, err := crd.GenerateYAML(crd.TemplateWebhookTemplate, []testkube.WebhookTemplate{result})
			return apiutils.SendLegacyCRDs(c, data, err)
		}

		return c.JSON(result)
	}
}

func (s *TestkubeAPI) DeleteWebhookTemplateHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		errPrefix := fmt.Sprintf("failed to delete webhook template %s", name)

		err := s.WebhookTemplatesClient.Delete(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: webhook template not found: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete webhook template: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func (s *TestkubeAPI) DeleteWebhookTemplatesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete webhook templates"

		err := s.WebhookTemplatesClient.DeleteByLabels(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete webhook templates: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}
