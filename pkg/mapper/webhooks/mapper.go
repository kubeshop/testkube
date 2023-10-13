package webhooks

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapCRDToAPI maps Webhook CRD to OpenAPI spec Webhook
func MapCRDToAPI(item executorv1.Webhook) testkube.Webhook {
	return testkube.Webhook{
		Name:                     item.Name,
		Namespace:                item.Namespace,
		Uri:                      item.Spec.Uri,
		Events:                   MapEventArrayToCRDEvents(item.Spec.Events),
		Selector:                 item.Spec.Selector,
		Labels:                   item.Labels,
		PayloadObjectField:       item.Spec.PayloadObjectField,
		PayloadTemplate:          item.Spec.PayloadTemplate,
		PayloadTemplateReference: item.Spec.PayloadTemplateReference,
		Headers:                  item.Spec.Headers,
	}
}

// MapStringArrayToCRDEvents maps string array of event types to OpenAPI spec list of EventType
func MapStringArrayToCRDEvents(items []string) (events []testkube.EventType) {
	for _, e := range items {
		events = append(events, testkube.EventType(e))
	}
	return
}

// MapEventArrayToCRDEvents maps event array of event types to OpenAPI spec list of EventType
func MapEventArrayToCRDEvents(items []executorv1.EventType) (events []testkube.EventType) {
	for _, e := range items {
		events = append(events, testkube.EventType(e))
	}
	return
}

// MapAPIToCRD maps OpenAPI spec WebhookCreateRequest to CRD Webhook
func MapAPIToCRD(request testkube.WebhookCreateRequest) executorv1.Webhook {
	return executorv1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: executorv1.WebhookSpec{
			Uri:                      request.Uri,
			Events:                   MapEventTypesToStringArray(request.Events),
			Selector:                 request.Selector,
			PayloadObjectField:       request.PayloadObjectField,
			PayloadTemplate:          request.PayloadTemplate,
			PayloadTemplateReference: request.PayloadTemplateReference,
			Headers:                  request.Headers,
		},
	}
}

// MapEventTypesToStringArray maps OpenAPI spec list of EventType to string array
func MapEventTypesToStringArray(eventTypes []testkube.EventType) (arr []executorv1.EventType) {
	for _, et := range eventTypes {
		arr = append(arr, executorv1.EventType(et))
	}
	return
}

// MapUpdateToSpec maps WebhookUpdateRequest to Wehook CRD spec
func MapUpdateToSpec(request testkube.WebhookUpdateRequest, webhook *executorv1.Webhook) *executorv1.Webhook {
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			request.Name,
			&webhook.Name,
		},
		{
			request.Namespace,
			&webhook.Namespace,
		},
		{
			request.Uri,
			&webhook.Spec.Uri,
		},
		{
			request.Selector,
			&webhook.Spec.Selector,
		},
		{
			request.PayloadObjectField,
			&webhook.Spec.PayloadObjectField,
		},
		{
			request.PayloadTemplate,
			&webhook.Spec.PayloadTemplate,
		},
		{
			request.PayloadTemplateReference,
			&webhook.Spec.PayloadTemplateReference,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
		}
	}

	if request.Events != nil {
		webhook.Spec.Events = MapEventTypesToStringArray(*request.Events)
	}

	if request.Labels != nil {
		webhook.Labels = *request.Labels
	}

	if request.Headers != nil {
		webhook.Spec.Headers = *request.Headers
	}

	return webhook
}

// MapSpecToUpdate maps Webhook CRD to WebhookUpdate Request to spec
func MapSpecToUpdate(webhook *executorv1.Webhook) (request testkube.WebhookUpdateRequest) {
	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&webhook.Name,
			&request.Name,
		},
		{
			&webhook.Namespace,
			&request.Namespace,
		},
		{
			&webhook.Spec.Uri,
			&request.Uri,
		},
		{
			&webhook.Spec.Selector,
			&request.Selector,
		},
		{
			&webhook.Spec.PayloadObjectField,
			&request.PayloadObjectField,
		},
		{
			&webhook.Spec.PayloadTemplate,
			&request.PayloadTemplate,
		},
		{
			&webhook.Spec.PayloadTemplateReference,
			&request.PayloadTemplateReference,
		},
	}

	for _, field := range fields {
		*field.destination = field.source
	}

	events := MapEventArrayToCRDEvents(webhook.Spec.Events)
	request.Events = &events

	request.Labels = &webhook.Labels
	request.Headers = &webhook.Spec.Headers

	return request
}
