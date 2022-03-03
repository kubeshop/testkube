package tests

import (
	testsv2 "github.com/kubeshop/testkube-operator/apis/tests/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTestListKubeToAPI(crTests testsv2.TestList) (tests []testkube.Test) {
	for _, item := range crTests.Items {
		tests = append(tests, MapTestCRToAPI(item))
	}

	return
}
func MapTestCRToAPI(crTest testsv2.Test) (test testkube.Test) {
	test.Name = crTest.Name
	test.Content = MapTestContentFromSpec(crTest.Spec.Content)
	test.Created = crTest.CreationTimestamp.Time
	test.Type_ = crTest.Spec.Type_
	test.Labels = crTest.Labels
	return
}

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
