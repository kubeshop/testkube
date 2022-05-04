package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	webhooksmapper "github.com/kubeshop/testkube/pkg/mapper/webhooks"
)

func (s TestkubeAPI) CreateWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.WebhookCreateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		webhook := webhooksmapper.MapAPIToCRD(request)
		webhook.Namespace = s.Namespace

		created, err := s.WebhooksClient.Create(&webhook)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(201)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) ListWebhooksHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		list, err := s.WebhooksClient.List(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		results := testkube.Webhooks{}
		for _, item := range list.Items {
			results = append(results, webhooksmapper.MapCRDToAPI(item))

		}
		return c.JSON(results)
	}
}

func (s TestkubeAPI) GetWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")

		item, err := s.WebhooksClient.Get(name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}
		result := webhooksmapper.MapCRDToAPI(*item)

		return c.JSON(result)
	}
}

func (s TestkubeAPI) DeleteWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")

		err := s.WebhooksClient.Delete(name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(204)
		return nil
	}
}

func (s TestkubeAPI) DeleteWebhooksHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := s.WebhooksClient.DeleteByLabels(c.Query("selector"))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(http.StatusNoContent)
		return nil
	}
}
