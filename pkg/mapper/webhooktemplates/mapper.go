package webhooktemplates

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapCRDToAPI maps WebhookTemplate CRD to OpenAPI spec WebhookTemplate
func MapCRDToAPI(item executorv1.WebhookTemplate) testkube.WebhookTemplate {
	return testkube.WebhookTemplate{
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

// MapAPIToCRD maps OpenAPI spec WebhookTemplateCreateRequest to CRD WebhookTemplate
func MapAPIToCRD(request testkube.WebhookTemplateCreateRequest) executorv1.WebhookTemplate {
	return executorv1.WebhookTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: executorv1.WebhookTemplateSpec{
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

// MapEventTypesToStringArray maps OpenAPI spec list of EventType to string array
func MapEventTypesToStringArray(eventTypes []testkube.EventType) (arr []executorv1.EventType) {
	for _, et := range eventTypes {
		arr = append(arr, executorv1.EventType(et))
	}
	return
}

// MapUpdateToSpec maps WebhookTemplateUpdateRequest to WebhookTemplate CRD spec
func MapUpdateToSpec(request testkube.WebhookTemplateUpdateRequest, webhookTemplate *executorv1.WebhookTemplate) *executorv1.WebhookTemplate {
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			request.Name,
			&webhookTemplate.Name,
		},
		{
			request.Namespace,
			&webhookTemplate.Namespace,
		},
		{
			request.Uri,
			&webhookTemplate.Spec.Uri,
		},
		{
			request.Selector,
			&webhookTemplate.Spec.Selector,
		},
		{
			request.PayloadObjectField,
			&webhookTemplate.Spec.PayloadObjectField,
		},
		{
			request.PayloadTemplate,
			&webhookTemplate.Spec.PayloadTemplate,
		},
		{
			request.PayloadTemplateReference,
			&webhookTemplate.Spec.PayloadTemplateReference,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
		}
	}

	if request.Events != nil {
		webhookTemplate.Spec.Events = MapEventTypesToStringArray(*request.Events)
	}

	if request.Labels != nil {
		webhookTemplate.Labels = *request.Labels
	}

	if request.Annotations != nil {
		webhookTemplate.Annotations = *request.Annotations
	}

	if request.Headers != nil {
		webhookTemplate.Spec.Headers = *request.Headers
	}

	if request.Disabled != nil {
		webhookTemplate.Spec.Disabled = *request.Disabled
	}

	if request.Config != nil {
		webhookTemplate.Spec.Config = common.MapMap(*request.Config, MapConfigValueAPIToCRD)
	}

	if request.Parameters != nil {
		webhookTemplate.Spec.Parameters = common.MapMap(*request.Parameters, MapParameterSchemaAPIToCRD)
	}

	return webhookTemplate
}

// MapSpecToUpdate maps WebhookTemplate CRD to WebhookTemplateUpdate Request to spec
func MapSpecToUpdate(webhookTemplate *executorv1.WebhookTemplate) (request testkube.WebhookTemplateUpdateRequest) {
	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&webhookTemplate.Name,
			&request.Name,
		},
		{
			&webhookTemplate.Namespace,
			&request.Namespace,
		},
		{
			&webhookTemplate.Spec.Uri,
			&request.Uri,
		},
		{
			&webhookTemplate.Spec.Selector,
			&request.Selector,
		},
		{
			&webhookTemplate.Spec.PayloadObjectField,
			&request.PayloadObjectField,
		},
		{
			&webhookTemplate.Spec.PayloadTemplate,
			&request.PayloadTemplate,
		},
		{
			&webhookTemplate.Spec.PayloadTemplateReference,
			&request.PayloadTemplateReference,
		},
	}

	for _, field := range fields {
		*field.destination = field.source
	}

	events := MapEventArrayToCRDEvents(webhookTemplate.Spec.Events)
	request.Events = &events

	request.Labels = &webhookTemplate.Labels
	request.Annotations = &webhookTemplate.Annotations
	request.Headers = &webhookTemplate.Spec.Headers
	request.Disabled = &webhookTemplate.Spec.Disabled
	request.Config = common.Ptr(common.MapMap(webhookTemplate.Spec.Config, MapConfigValueCRDToAPI))
	request.Parameters = common.Ptr(common.MapMap(webhookTemplate.Spec.Parameters, MapParameterSchemaCRDToAPI))

	return request
}
