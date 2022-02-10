package tests

import (
	testsv2 "github.com/kubeshop/testkube-operator/apis/tests/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MapScriptToScriptSpec(request testkube.TestUpsertRequest) *testsv2.Test {

	test := &testsv2.Test{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
		},
		Spec: testsv2.TestSpec{
			Type_:   request.Type_,
			Content: MapScriptContentToScriptSpecContent(request.Content),
			Tags:    request.Tags,
		},
	}

	return test

}

func MapScriptContentToScriptSpecContent(content *testkube.TestContent) (specContent *testsv2.TestContent) {
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
