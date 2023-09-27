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
	webhooksmapper "github.com/kubeshop/testkube/pkg/mapper/webhooks"
)

func (s TestkubeAPI) CreateWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create webhook"
		var webhook executorv1.Webhook
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			webhookSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(webhookSpec), len(webhookSpec))
			if err := decoder.Decode(&webhook); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}
		} else {
			var request testkube.WebhookCreateRequest
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}

			if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
				if request.PayloadTemplate != "" {
					request.PayloadTemplate = fmt.Sprintf("%q", request.PayloadTemplate)
				}

				data, err := crd.GenerateYAML(crd.TemplateWebhook, []testkube.WebhookCreateRequest{request})
				return s.getCRDs(c, data, err)
			}

			webhook = webhooksmapper.MapAPIToCRD(request)
			webhook.Namespace = s.Namespace
		}

		created, err := s.WebhooksClient.Create(&webhook)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create webhook: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) UpdateWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to update webhook"
		var request testkube.WebhookUpdateRequest
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			var webhook executorv1.Webhook
			webhookSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(webhookSpec), len(webhookSpec))
			if err := decoder.Decode(&webhook); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}

			request = webhooksmapper.MapSpecToUpdate(&webhook)
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
		webhook, err := s.WebhooksClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client found no webhook: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get webhook: %w", errPrefix, err))
		}

		// map update webhook but load spec only to not override metadata.ResourceVersion
		webhookSpec := webhooksmapper.MapUpdateToSpec(request, webhook)

		updatedWebhook, err := s.WebhooksClient.Update(webhookSpec)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update webhook: %w", errPrefix, err))
		}

		return c.JSON(updatedWebhook)
	}
}

func (s TestkubeAPI) ListWebhooksHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list webhooks"

		list, err := s.WebhooksClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list webhooks: %w", errPrefix, err))
		}

		results := testkube.Webhooks{}
		for _, item := range list.Items {
			results = append(results, webhooksmapper.MapCRDToAPI(item))

		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range results {
				if results[i].PayloadTemplate != "" {
					results[i].PayloadTemplate = fmt.Sprintf("%q", results[i].PayloadTemplate)
				}
			}

			data, err := crd.GenerateYAML(crd.TemplateWebhook, results)
			return s.getCRDs(c, data, err)
		}

		return c.JSON(results)
	}
}

func (s TestkubeAPI) GetWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		errPrefix := fmt.Sprintf("failed to get webhook %s", name)

		item, err := s.WebhooksClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: webhook not found: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get webhook: %w", errPrefix, err))
		}

		result := webhooksmapper.MapCRDToAPI(*item)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			if result.PayloadTemplate != "" {
				result.PayloadTemplate = fmt.Sprintf("%q", result.PayloadTemplate)
			}

			data, err := crd.GenerateYAML(crd.TemplateWebhook, []testkube.Webhook{result})
			return s.getCRDs(c, data, err)
		}

		return c.JSON(result)
	}
}

func (s TestkubeAPI) DeleteWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		errPrefix := fmt.Sprintf("failed to delete webhook %s", name)

		err := s.WebhooksClient.Delete(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: webhook not found: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete webhook: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}

func (s TestkubeAPI) DeleteWebhooksHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete webhooks"

		err := s.WebhooksClient.DeleteByLabels(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete webhooks: %w", errPrefix, err))
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}
