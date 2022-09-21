package testtriggers

import (
	testsv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MapTestTriggerUpsertRequestToTestTriggerCRD(request testkube.TestTriggerUpsertRequest) testsv1.TestTrigger {
	return testsv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsv1.TestTriggerSpec{
			Resource:         request.Resource,
			ResourceSelector: mapTestTriggerUpsertRequestSelectorToTestTriggerSelectorCRD(request.ResourceSelector),
			Event:            request.Event,
			Action:           request.Action,
			Execution:        request.Execution,
			TestSelector:     mapTestTriggerUpsertRequestSelectorToTestTriggerSelectorCRD(request.TestSelector),
		},
	}
}

func mapTestTriggerUpsertRequestSelectorToTestTriggerSelectorCRD(selector *testkube.TestTriggerSelector) testsv1.TestTriggerSelector {
	return testsv1.TestTriggerSelector{
		Name:      selector.Name,
		Namespace: selector.Namespace,
		Labels:    selector.Labels,
	}
}
