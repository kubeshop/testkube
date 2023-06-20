package testtriggers

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
func MapCRDToAPI(crd *testsv1.TestTrigger) testkube.TestTrigger {
	resource := testkube.TestTriggerResources(crd.Spec.Resource)
	action := testkube.TestTriggerActions(crd.Spec.Action)
	execution := testkube.TestTriggerExecutions(crd.Spec.Execution)

	return testkube.TestTrigger{
		Name:             crd.Name,
		Namespace:        crd.Namespace,
		Labels:           crd.Labels,
		Resource:         &resource,
		ResourceSelector: mapSelectorFromCRD(crd.Spec.ResourceSelector),
		Event:            string(crd.Spec.Event),
		ConditionSpec:    mapConditionSpecFromCRD(crd.Spec.ConditionSpec),
		Action:           &action,
		Execution:        &execution,
		TestSelector:     mapSelectorFromCRD(crd.Spec.TestSelector),
	}
}

func mapSelectorFromCRD(selector testsv1.TestTriggerSelector) *testkube.TestTriggerSelector {
	var labelSelector *testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector
	if selector.LabelSelector != nil {
		labelSelector = mapLabelSelectorFromCRD(selector.LabelSelector)
	}
	return &testkube.TestTriggerSelector{
		Name:          selector.Name,
		Namespace:     selector.Namespace,
		LabelSelector: labelSelector,
	}
}

func mapLabelSelectorFromCRD(labelSelector *v1.LabelSelector) *testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector {
	var matchExpressions []testkube.IoK8sApimachineryPkgApisMetaV1LabelSelectorRequirement
	for _, e := range labelSelector.MatchExpressions {
		expression := testkube.IoK8sApimachineryPkgApisMetaV1LabelSelectorRequirement{
			Key:      e.Key,
			Operator: string(e.Operator),
			Values:   e.Values,
		}
		matchExpressions = append(matchExpressions, expression)
	}

	return &testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector{
		MatchExpressions: matchExpressions,
		MatchLabels:      labelSelector.MatchLabels,
	}
}

func mapConditionSpecFromCRD(conditionSpec *testsv1.TestTriggerConditionSpec) *testkube.TestTriggerConditionSpec {
	if conditionSpec == nil {
		return nil
	}

	var conditions []testkube.TestTriggerCondition
	for _, condition := range conditionSpec.Conditions {
		conditions = append(conditions, testkube.TestTriggerCondition{
			Type_:  condition.Type_,
			Status: (*testkube.TestTriggerConditionStatuses)(condition.Status),
			Reason: condition.Reason,
			Ttl:    condition.Ttl,
		})
	}

	return &testkube.TestTriggerConditionSpec{
		Timeout:    conditionSpec.Timeout,
		Conditions: conditions,
	}
}

func MapTestTriggerCRDToTestTriggerUpsertRequest(request testsv1.TestTrigger) testkube.TestTriggerUpsertRequest {
	return testkube.TestTriggerUpsertRequest{
		Name:             request.Name,
		Namespace:        request.Namespace,
		Labels:           request.Labels,
		Resource:         (*testkube.TestTriggerResources)(&request.Spec.Resource),
		ResourceSelector: mapSelectorFromCRD(request.Spec.ResourceSelector),
		Event:            string(request.Spec.Event),
		ConditionSpec:    mapConditionSpecFromCRD(request.Spec.ConditionSpec),
		Action:           (*testkube.TestTriggerActions)(&request.Spec.Action),
		Execution:        (*testkube.TestTriggerExecutions)(&request.Spec.Execution),
		TestSelector:     mapSelectorFromCRD(request.Spec.TestSelector),
	}
}
