package webhooks

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapCRDToAPI maps Webhook CRD to OpenAPI spec Webhook
func MapCRDToAPI(item executorv1.Webhook) testkube.Webhook {
	return testkube.Webhook{
		Name:               item.Name,
		Namespace:          item.Namespace,
		Uri:                item.Spec.Uri,
		Events:             MapEventArrayToCRDEvents(item.Spec.Events),
		Selector:           item.Spec.Selector,
		Labels:             item.Labels,
		PayloadObjectField: item.Spec.PayloadObjectField,
		PayloadTemplate:    item.Spec.PayloadTemplate,
		Headers:            item.Spec.Headers,
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
			Uri:                request.Uri,
			Events:             MapEventTypesToStringArray(request.Events),
			Selector:           request.Selector,
			PayloadObjectField: request.PayloadObjectField,
			PayloadTemplate:    request.PayloadTemplate,
			Headers:            request.Headers,
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
