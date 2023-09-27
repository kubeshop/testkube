package templates

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	templatev1 "github.com/kubeshop/testkube-operator/api/template/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapCRDToAPI maps Template CRD to OpenAPI spec Template
func MapCRDToAPI(item templatev1.Template) testkube.Template {
	return testkube.Template{
		Name:      item.Name,
		Namespace: item.Namespace,
		Body:      item.Spec.Body,
		Type_:     (*testkube.TemplateType)(item.Spec.Type_),
		Labels:    item.Labels,
	}
}

// MapAPIToCRD maps OpenAPI spec TemplateCreateRequest to CRD Template
func MapAPIToCRD(request testkube.TemplateCreateRequest) templatev1.Template {
	return templatev1.Template{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: templatev1.TemplateSpec{
			Type_: (*templatev1.TemplateType)(request.Type_),
			Body:  request.Body,
		},
	}
}

// MapUpdateToSpec maps TemplateUpdateRequest to Wehook CRD spec
func MapUpdateToSpec(request testkube.TemplateUpdateRequest, template *templatev1.Template) *templatev1.Template {
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			request.Name,
			&template.Name,
		},
		{
			request.Namespace,
			&template.Namespace,
		},
		{
			request.Body,
			&template.Spec.Body,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
		}
	}

	if request.Type_ != nil {
		*template.Spec.Type_ = (templatev1.TemplateType)(*request.Type_)
	}

	if request.Labels != nil {
		template.Labels = *request.Labels
	}

	return template
}

// MapSpecToUpdate maps Template CRD to TemplateUpdate Request to spec
func MapSpecToUpdate(template *templatev1.Template) (request testkube.TemplateUpdateRequest) {
	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&template.Name,
			&request.Name,
		},
		{
			&template.Namespace,
			&request.Namespace,
		},
		{
			&template.Spec.Body,
			&request.Body,
		},
	}

	for _, field := range fields {
		*field.destination = field.source
	}

	request.Type_ = (*testkube.TemplateType)(template.Spec.Type_)
	request.Labels = &template.Labels

	return request
}
