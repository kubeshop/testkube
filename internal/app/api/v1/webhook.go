package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s TestkubeAPI) CreateWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.WebhookCreateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		executor := mapWebhookCreateRequestToWebhookCRD(request)
		created, err := s.WebhookClient.Create(&executor)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Status(201)
		return c.JSON(created)
	}
}

func (s TestkubeAPI) ListWebhooksHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ns := c.Query("namespace", "testkube")
		list, err := s.WebhookClient.List(ns)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		results := []testkube.Webhook{}
		for _, item := range list.Items {
			results = append(results, mapWebhookCRDToWebhook(item))

		}
		return c.JSON(results)
	}
}

func (s TestkubeAPI) GetWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		ns := c.Query("namespace", "testkube")

		item, err := s.WebhookClient.Get(ns, name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}
		result := mapWebhookCRDToWebhook(*item)

		return c.JSON(result)
	}
}

func (s TestkubeAPI) DeleteWebhookHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		ns := c.Query("namespace", "testkube")

		err := s.WebhookClient.Delete(name, ns)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		c.Context().SetStatusCode(204)
		return nil
	}
}

func mapWebhookCRDToWebhook(item executorv1.Webhook) testkube.Webhook {
	return testkube.Webhook{
		Name:      item.Name,
		Namespace: item.Namespace,
		Uri:       item.Spec.Uri,
		Events:    mapWebhookCRDEventsToWebhookEvents(item.Spec.Events),
	}
}

func mapWebhookCRDEventsToWebhookEvents(items []string) (events []testkube.WebhookEventType) {
	for _, e := range items {
		events = append(events, testkube.WebhookEventType(e))
	}
	return
}

func mapWebhookCreateRequestToWebhookCRD(request testkube.WebhookCreateRequest) executorv1.Webhook {
	return executorv1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
		},
		Spec: executorv1.WebhookSpec{
			Uri:    request.Uri,
			Events: mapWebhookEventTypesToArrayString(request.Events),
		},
	}
}

func mapWebhookEventTypesToArrayString(eventTypes []testkube.WebhookEventType) (arr []string) {
	for _, et := range eventTypes {
		arr = append(arr, string(et))
	}
	return
}
