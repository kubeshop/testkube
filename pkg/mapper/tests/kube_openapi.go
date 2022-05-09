package tests

import (
	testsv2 "github.com/kubeshop/testkube-operator/apis/tests/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapTestListKubeToAPI maps CRD list data to OpenAPI spec tests list
func MapTestListKubeToAPI(crTests testsv2.TestList) (tests []testkube.Test) {
	tests = []testkube.Test{}
	for _, item := range crTests.Items {
		tests = append(tests, MapTestCRToAPI(item))
	}

	return
}

// MapTestCRToAPI maps CRD to OpenAPI spec test
func MapTestCRToAPI(crTest testsv2.Test) (test testkube.Test) {
	test.Name = crTest.Name
	test.Content = MapTestContentFromSpec(crTest.Spec.Content)
	test.Created = crTest.CreationTimestamp.Time
	test.Type_ = crTest.Spec.Type_
	test.Labels = crTest.Labels
	test.Variables = MapVariables(crTest.Spec.Params)
	test.Schedule = crTest.Spec.Schedule
	return
}

func MapVariables(in map[string]string) map[string]testkube.Variable {
	out := map[string]testkube.Variable{}
	for k, v := range in {
		out[k] = testkube.NewBasicVariable(k, v)
	}
	return out
}

// MapTestContentFromSpec maps CRD to OpenAPI spec TestContent
func MapTestContentFromSpec(specContent *testsv2.TestContent) *testkube.TestContent {
	content := &testkube.TestContent{
		Type_: specContent.Type_,
		// assuming same data structure - there is task about syncing them automatically
		// https://github.com/kubeshop/testkube/issues/723
		Repository: (*testkube.Repository)(specContent.Repository),
		Data:       specContent.Data,
		Uri:        specContent.Uri,
	}

	return content
}
