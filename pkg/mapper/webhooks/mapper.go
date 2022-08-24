package webhooks

import (
	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MapCRDToAPI maps Webhook CRD to OpenAPI spec Webhook
func MapCRDToAPI(item executorv1.Webhook) testkube.Webhook {
	return testkube.Webhook{
		Name:      item.Name,
		Namespace: item.Namespace,
		Uri:       item.Spec.Uri,
		Events:    MapStringArrayToCRDEvents(item.Spec.Events),
		Selector:  item.Spec.Selector,
		Labels:    item.Labels,
	}
}

// MapStringArrayToCRDEvents maps string array of event types to OpenAPI spec list of TestkubeEventType
func MapStringArrayToCRDEvents(items []string) (events []testkube.TestkubeEventType) {
	for _, e := range items {
		events = append(events, testkube.TestkubeEventType(e))
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
			Uri:      request.Uri,
			Events:   MapEventTypesToStringArray(request.Events),
			Selector: request.Selector,
		},
	}
}

// MapEventTypesToStringArray maps OpenAPI spec list of TestkubeEventType to string array
func MapEventTypesToStringArray(eventTypes []testkube.TestkubeEventType) (arr []string) {
	for _, et := range eventTypes {
		arr = append(arr, string(et))
	}
	return
}
