package tests

import (
	testsv2 "github.com/kubeshop/testkube-operator/apis/tests/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MapToSpec maps TestUpsertRequest to Test CRD spec
func MapToSpec(request testkube.TestUpsertRequest) *testsv2.Test {

	test := &testsv2.Test{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsv2.TestSpec{
			Type_:     request.Type_,
			Content:   MapContentToSpecContent(request.Content),
			Schedule:  request.Schedule,
			Variables: MapCRDVariables(request.Variables),
		},
	}

	return test

}

// @Depracated
// MapDepratcatedParams maps old params to new variables data structure
func MapDepratcatedParams(in map[string]testkube.Variable) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v.Value
	}
	return out
}

// MapCRDVariables maps variables between API and operator CRDs
func MapCRDVariables(in map[string]testkube.Variable) map[string]testsv2.Variable {
	out := map[string]testsv2.Variable{}
	for k, v := range in {
		out[k] = testsv2.Variable{
			Name:  v.Name,
			Type_: string(*v.Type_),
			Value: v.Value,
		}
	}
	return out
}

// MapContentToSpecContent maps TestContent OpenAPI spec to TestContent CRD spec
func MapContentToSpecContent(content *testkube.TestContent) (specContent *testsv2.TestContent) {
	if content == nil {
		return
	}

	return &testsv2.TestContent{
		// assuming same data structure
		Repository: (*testsv2.Repository)(content.Repository),
		Data:       content.Data,
		Uri:        content.Uri,
		Type_:      content.Type_,
	}
}
