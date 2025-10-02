package testtriggers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
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
			Selector:          mapLabelSelectorToCRD(request.Selector),
			Resource:          resource,
			ResourceSelector:  mapSelectorToCRD(request.ResourceSelector),
			Event:             testsv1.TestTriggerEvent(request.Event),
			ConditionSpec:     mapConditionSpecCRD(request.ConditionSpec),
			ProbeSpec:         mapProbeSpecCRD(request.ProbeSpec),
			Action:            action,
			ActionParameters:  mapActionParametersCRD(request.ActionParameters),
			Execution:         execution,
			TestSelector:      mapSelectorToCRD(request.TestSelector),
			ConcurrencyPolicy: concurrencyPolicy,
			Disabled:          request.Disabled,
		},
	}
}

// MapTestTriggerUpsertRequestToTestTriggerCRDWithExistingMeta creates a TestTrigger CRD from an upsert request
// while preserving the existing ObjectMeta (including ResourceVersion) from the original CRD
func MapTestTriggerUpsertRequestToTestTriggerCRDWithExistingMeta(request testkube.TestTriggerUpsertRequest, existingMeta metav1.ObjectMeta) testsv1.TestTrigger {
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

	// Preserve existing metadata but update labels and annotations
	updatedMeta := existingMeta.DeepCopy()
	if request.Labels != nil {
		updatedMeta.Labels = request.Labels
	}
	if request.Annotations != nil {
		updatedMeta.Annotations = request.Annotations
	}

	return testsv1.TestTrigger{
		ObjectMeta: *updatedMeta,
		Spec: testsv1.TestTriggerSpec{
			Selector:          mapLabelSelectorToCRD(request.Selector),
			Resource:          resource,
			ResourceSelector:  mapSelectorToCRD(request.ResourceSelector),
			Event:             testsv1.TestTriggerEvent(request.Event),
			ConditionSpec:     mapConditionSpecCRD(request.ConditionSpec),
			ProbeSpec:         mapProbeSpecCRD(request.ProbeSpec),
			Action:            action,
			ActionParameters:  mapActionParametersCRD(request.ActionParameters),
			Execution:         execution,
			TestSelector:      mapSelectorToCRD(request.TestSelector),
			ConcurrencyPolicy: concurrencyPolicy,
			Disabled:          request.Disabled,
		},
	}
}

func mapSelectorToCRD(selector *testkube.TestTriggerSelector) testsv1.TestTriggerSelector {
	return testsv1.TestTriggerSelector{
		Name:           selector.Name,
		NameRegex:      selector.NameRegex,
		Namespace:      selector.Namespace,
		NamespaceRegex: selector.NamespaceRegex,
		LabelSelector:  mapLabelSelectorToCRD(selector.LabelSelector),
	}
}

func mapLabelSelectorToCRD(labelSelector *testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}
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

func mapActionParametersCRD(actionParameters *testkube.TestTriggerActionParameters) *testsv1.TestTriggerActionParameters {
	if actionParameters == nil {
		return nil
	}

	return &testsv1.TestTriggerActionParameters{
		Config: actionParameters.Config,
		Tags:   actionParameters.Tags,
		Target: common.MapPtr(actionParameters.Target, commonmapper.MapTargetApiToKube),
	}
}
