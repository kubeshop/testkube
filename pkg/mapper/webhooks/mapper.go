package webhooks

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	"github.com/kubeshop/testkube/internal/common"
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
		Disabled:                 item.Spec.Disabled,
		Config:                   common.MapMap(item.Spec.Config, MapConfigValueCRDToAPI),
		Parameters:               common.MapMap(item.Spec.Parameters, MapParameterSchemaCRDToAPI),
		WebhookTemplateRef:       common.MapPtr(item.Spec.WebhookTemplateRef, MapTemplateRefCRDToAPI),
		IsTemplate:               item.Spec.IsTemplate,
	}
}

// MapStringToBoxedString maps string to boxed string
func MapStringToBoxedString(v *string) *testkube.BoxedString {
	if v == nil {
		return nil
	}
	return &testkube.BoxedString{Value: *v}
}

// MapSecretRefCRDToAPI maps secret ref to OpenAPI spec
func MapSecretRefCRDToAPI(v executorv1.SecretRef) testkube.SecretRef {
	return testkube.SecretRef{
		Namespace: v.Namespace,
		Name:      v.Name,
		Key:       v.Key,
	}
}

// MapConigValueCRDToAPI maps config value to OpenAPI spec
func MapConfigValueCRDToAPI(v executorv1.WebhookConfigValue) testkube.WebhookConfigValue {
	return testkube.WebhookConfigValue{
		Value:  MapStringToBoxedString(v.Value),
		Secret: common.MapPtr(v.Secret, MapSecretRefCRDToAPI),
	}
}

// MapParameterSchemaCRDToAPI maps parameter schema to OpenAPI spec
func MapParameterSchemaCRDToAPI(v executorv1.WebhookParameterSchema) testkube.WebhookParameterSchema {
	return testkube.WebhookParameterSchema{
		Description: v.Description,
		Required:    v.Required,
		Example:     v.Example,
		Default_:    MapStringToBoxedString(v.Default_),
		Pattern:     v.Pattern,
	}
}

// MapTemplateRefCRDToAPI maps template ref to OpenAPI spec
func MapTemplateRefCRDToAPI(v executorv1.WebhookTemplateRef) testkube.WebhookTemplateRef {
	return testkube.WebhookTemplateRef{
		Name: v.Name,
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
			Disabled:                 request.Disabled,
			Config:                   common.MapMap(request.Config, MapConfigValueAPIToCRD),
			Parameters:               common.MapMap(request.Parameters, MapParameterSchemaAPIToCRD),
			WebhookTemplateRef:       common.MapPtr(request.WebhookTemplateRef, MapTemplateRefAPIToCRD),
			IsTemplate:               request.IsTemplate,
		},
	}
}

// MapBoxedStringToString maps boxed string to string
func MapBoxedStringToString(v *testkube.BoxedString) *string {
	if v == nil {
		return nil
	}
	return &v.Value
}

// MapSecretRefAPIToCRD maps secret ref to CRD spec
func MapSecretRefAPIToCRD(v testkube.SecretRef) executorv1.SecretRef {
	return executorv1.SecretRef{
		Namespace: v.Namespace,
		Name:      v.Name,
		Key:       v.Key,
	}
}

// MapConigValueAPIToCRD maps config value to CRD spec
func MapConfigValueAPIToCRD(v testkube.WebhookConfigValue) executorv1.WebhookConfigValue {
	return executorv1.WebhookConfigValue{
		Value:  MapBoxedStringToString(v.Value),
		Secret: common.MapPtr(v.Secret, MapSecretRefAPIToCRD),
	}
}

// MapParameterSchemaAPIToCRD maps parameter schema to CRD spec
func MapParameterSchemaAPIToCRD(v testkube.WebhookParameterSchema) executorv1.WebhookParameterSchema {
	return executorv1.WebhookParameterSchema{
		Description: v.Description,
		Required:    v.Required,
		Example:     v.Example,
		Default_:    MapBoxedStringToString(v.Default_),
		Pattern:     v.Pattern,
	}
}

// MapTemplateRefAPIToCRD maps template ref to CRD spec
func MapTemplateRefAPIToCRD(v testkube.WebhookTemplateRef) executorv1.WebhookTemplateRef {
	return executorv1.WebhookTemplateRef{
		Name: v.Name,
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

	if request.Annotations != nil {
		webhook.Annotations = *request.Annotations
	}

	if request.Headers != nil {
		webhook.Spec.Headers = *request.Headers
	}

	if request.Disabled != nil {
		webhook.Spec.Disabled = *request.Disabled
	}

	if request.Config != nil {
		webhook.Spec.Config = common.MapMap(*request.Config, MapConfigValueAPIToCRD)
	}

	if request.Parameters != nil {
		webhook.Spec.Parameters = common.MapMap(*request.Parameters, MapParameterSchemaAPIToCRD)
	}

	if request.WebhookTemplateRef != nil {
		webhook.Spec.WebhookTemplateRef = common.MapPtr(*request.WebhookTemplateRef, MapTemplateRefAPIToCRD)
	}

	if request.IsTemplate != nil {
		webhook.Spec.IsTemplate = *request.IsTemplate
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
	request.Annotations = &webhook.Annotations
	request.Headers = &webhook.Spec.Headers
	request.Disabled = &webhook.Spec.Disabled
	request.Config = common.Ptr(common.MapMap(webhook.Spec.Config, MapConfigValueCRDToAPI))
	request.Parameters = common.Ptr(common.MapMap(webhook.Spec.Parameters, MapParameterSchemaCRDToAPI))
	request.WebhookTemplateRef = common.Ptr(common.MapPtr(webhook.Spec.WebhookTemplateRef, MapTemplateRefCRDToAPI))
	request.IsTemplate = &webhook.Spec.IsTemplate

	return request
}
