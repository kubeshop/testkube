package testtriggers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTestTriggerUpsertRequestToTestTriggerCRD(request testkube.TestTriggerUpsertRequest) testsv1.TestTrigger {
	var resource testsv1.TestTriggerResource
	if request.Resource != nil {
		resource = testsv1.TestTriggerResource(*request.Resource)
	}

	var action testsv1.TestTriggerAction
	if request.Action != nil {
		action = testsv1.TestTriggerAction(*request.Action)
	}

	var execution testsv1.TestTriggerExecution
	if request.Execution != nil {
		execution = testsv1.TestTriggerExecution(*request.Execution)
	}

	var concurrencyPolicy testsv1.TestTriggerConcurrencyPolicy
	if request.ConcurrencyPolicy != nil {
		concurrencyPolicy = testsv1.TestTriggerConcurrencyPolicy(*request.ConcurrencyPolicy)
	}

	return testsv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsv1.TestTriggerSpec{
			Resource:          resource,
			ResourceSelector:  mapSelectorToCRD(request.ResourceSelector),
			Event:             testsv1.TestTriggerEvent(request.Event),
			ConditionSpec:     mapConditionSpecCRD(request.ConditionSpec),
			ProbeSpec:         mapProbeSpecCRD(request.ProbeSpec),
			Action:            action,
			Execution:         execution,
			TestSelector:      mapSelectorToCRD(request.TestSelector),
			ConcurrencyPolicy: concurrencyPolicy,
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
		NameRegex:     selector.NameRegex,
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
			Ttl:    condition.Ttl,
		})
	}

	return &testsv1.TestTriggerConditionSpec{
		Timeout:    conditionSpec.Timeout,
		Delay:      conditionSpec.Delay,
		Conditions: conditions,
	}
}

func mapProbeSpecCRD(probeSpec *testkube.TestTriggerProbeSpec) *testsv1.TestTriggerProbeSpec {
	if probeSpec == nil {
		return nil
	}

	var probes []testsv1.TestTriggerProbe
	for _, probe := range probeSpec.Probes {
		var headers map[string]string
		if len(probe.Headers) != 0 {
			headers = make(map[string]string, len(probe.Headers))
			for key, value := range probe.Headers {
				headers[key] = value
			}
		}

		probes = append(probes, testsv1.TestTriggerProbe{
			Scheme:  probe.Scheme,
			Host:    probe.Host,
			Path:    probe.Path,
			Port:    probe.Port,
			Headers: headers,
		})
	}

	return &testsv1.TestTriggerProbeSpec{
		Timeout: probeSpec.Timeout,
		Delay:   probeSpec.Delay,
		Probes:  probes,
	}
}
