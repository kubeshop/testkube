package testtriggers

import (
	testsv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapTestTriggerListKubeToAPI maps TestTriggerList CRD to list of OpenAPI spec TestTrigger
func MapTestTriggerListKubeToAPI(crd *testsv1.TestTriggerList) (testTriggers []testkube.TestTrigger) {
	testTriggers = make([]testkube.TestTrigger, len(crd.Items))
	for i := range crd.Items {
		testTriggers[i] = MapCRDToAPI(&crd.Items[i])
	}

	return
}

// MapCRDToAPI maps TestTrigger CRD to OpenAPI spec TestTrigger
func MapCRDToAPI(crd *testsv1.TestTrigger) (testTrigger testkube.TestTrigger) {
	testTrigger.Name = crd.Name
	testTrigger.Namespace = crd.Namespace
	testTrigger.Labels = crd.Labels
	testTrigger.Resource = crd.Spec.Resource
	testTrigger.ResourceSelector = mapSelectorFromCRDSpec(crd.Spec.ResourceSelector)
	testTrigger.Event = crd.Spec.Event
	testTrigger.Action = crd.Spec.Action
	testTrigger.Execution = crd.Spec.Execution
	testTrigger.TestSelector = mapSelectorFromCRDSpec(crd.Spec.TestSelector)

	return
}

func mapSelectorFromCRDSpec(selector testsv1.TestTriggerSelector) *testkube.TestTriggerSelector {
	return &testkube.TestTriggerSelector{
		Name:      selector.Name,
		Namespace: selector.Namespace,
		Labels:    selector.Labels,
	}
}
