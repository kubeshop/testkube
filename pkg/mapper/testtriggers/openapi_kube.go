package testtriggers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTestTriggerUpsertRequestToTestTriggerCRD(request testkube.TestTriggerUpsertRequest) testsv1.TestTrigger {
	return testsv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsv1.TestTriggerSpec{
			Resource:         string(*request.Resource),
			ResourceSelector: mapSelectorToCRD(request.ResourceSelector),
			Event:            request.Event,
			ConditionSpec:    mapConditionSpecCRD(request.ConditionSpec),
			Action:           string(*request.Action),
			Execution:        string(*request.Execution),
			TestSelector:     mapSelectorToCRD(request.TestSelector),
		},
	}
}

func mapSelectorToCRD(selector *testkube.TestTriggerSelector) testsv1.TestTriggerSelector {
	var labelSelector *metav1.LabelSelector
	if selector.LabelSelector != nil {
		labelSelector = mapLabelSelectorToCRD(selector.LabelSelector)
	}
	return testsv1.TestTriggerSelector{
		Name:          selector.Name,
		Namespace:     selector.Namespace,
		LabelSelector: labelSelector,
	}
}

func mapLabelSelectorToCRD(labelSelector *testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector) *metav1.LabelSelector {
	var matchExpressions []metav1.LabelSelectorRequirement
	for _, e := range labelSelector.MatchExpressions {
		expression := metav1.LabelSelectorRequirement{
			Key:      e.Key,
			Operator: metav1.LabelSelectorOperator(e.Operator),
			Values:   e.Values,
		}
		matchExpressions = append(matchExpressions, expression)
	}

	return &metav1.LabelSelector{
		MatchLabels:      labelSelector.MatchLabels,
		MatchExpressions: matchExpressions,
	}
}

func mapConditionSpecCRD(conditionSpec *testkube.TestTriggerConditionSpec) *testsv1.TestTriggerConditionSpec {
	if conditionSpec == nil {
		return nil
	}

	var conditions []testsv1.TestTriggerCondition
	for _, condition := range conditionSpec.Conditions {
		conditions = append(conditions, testsv1.TestTriggerCondition{
			Type_:  condition.Type_,
			Status: (*testsv1.TestTriggerConditionStatuses)(condition.Status),
			Reason: condition.Reason,
		})
	}

	return &testsv1.TestTriggerConditionSpec{
		Timeout:    conditionSpec.Timeout,
		Conditions: conditions,
	}
}
