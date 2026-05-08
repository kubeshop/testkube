package testtriggers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	testsv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
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
			ResourceRef:       mapResourceRefToCRD(request.ResourceRef),
			ResourceSelector:  mapSelectorToCRD(request.ResourceSelector),
			Event:             testsv1.TestTriggerEvent(request.Event),
			Match:             mapFieldConditionsToCRD(request.Match),
			ConditionSpec:     mapConditionSpecCRD(request.ConditionSpec),
			ProbeSpec:         mapProbeSpecCRD(request.ProbeSpec),
			ContentSelector:   mapContentSelectorToCRD(request.ContentSelector),
			Action:            action,
			ActionParameters:  mapActionParametersCRD(request.ActionParameters),
			Execution:         execution,
			TestSelector:      mapSelectorToCRD(request.TestSelector),
			ConcurrencyPolicy: concurrencyPolicy,
			Disabled:          request.Disabled,
		},
	}
}

func mapResourceRefToCRD(ref *testkube.TestTriggerResourceRef) *testsv1.TestTriggerResourceRef {
	if ref == nil {
		return nil
	}
	return &testsv1.TestTriggerResourceRef{
		Group:   ref.Group,
		Version: ref.Version,
		Kind:    ref.Kind,
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
			ResourceRef:       mapResourceRefToCRD(request.ResourceRef),
			ResourceSelector:  mapSelectorToCRD(request.ResourceSelector),
			Event:             testsv1.TestTriggerEvent(request.Event),
			Match:             mapFieldConditionsToCRD(request.Match),
			ConditionSpec:     mapConditionSpecCRD(request.ConditionSpec),
			ProbeSpec:         mapProbeSpecCRD(request.ProbeSpec),
			ContentSelector:   mapContentSelectorToCRD(request.ContentSelector),
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

func mapFieldConditionsToCRD(in []testkube.TestTriggerFieldCondition) []workflowtriggersv1.WorkflowTriggerFieldCondition {
	if len(in) == 0 {
		return nil
	}
	out := make([]workflowtriggersv1.WorkflowTriggerFieldCondition, 0, len(in))
	for _, c := range in {
		out = append(out, workflowtriggersv1.WorkflowTriggerFieldCondition{
			Path:     c.Path,
			Operator: workflowtriggersv1.WorkflowTriggerFieldOperator(c.Operator),
			Value:    c.Value,
		})
	}
	return out
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

func mapContentSelectorToCRD(selector *testkube.TestTriggerContentSelector) *testsv1.TestTriggerContentSelector {
	if selector == nil {
		return nil
	}
	return &testsv1.TestTriggerContentSelector{
		Git: mapContentGitToCRD(selector.Git),
	}
}

func mapContentGitToCRD(git *testkube.TestTriggerContentGit) *testsv1.TestTriggerContentGitSpec {
	if git == nil {
		return nil
	}
	return &testsv1.TestTriggerContentGitSpec{
		Uri:          git.Uri,
		Revision:     git.Revision,
		Username:     git.Username,
		UsernameFrom: mapEnvVarSourceAPIToKube(git.UsernameFrom),
		Token:        git.Token,
		TokenFrom:    mapEnvVarSourceAPIToKube(git.TokenFrom),
		SshKey:       git.SshKey,
		SshKeyFrom:   mapEnvVarSourceAPIToKube(git.SshKeyFrom),
		AuthType:     mapGitAuthTypeAPIToKube(git.AuthType),
		MountPath:    git.MountPath,
		Cone:         git.Cone,
		Paths:        git.Paths,
	}
}

func mapEnvVarSourceAPIToKube(v *testkube.EnvVarSource) *corev1.EnvVarSource {
	if v == nil {
		return nil
	}
	return &corev1.EnvVarSource{
		FieldRef:         mapFieldRefAPIToKube(v.FieldRef),
		ResourceFieldRef: mapResourceFieldRefAPIToKube(v.ResourceFieldRef),
		ConfigMapKeyRef:  mapConfigMapKeyRefAPIToKube(v.ConfigMapKeyRef),
		SecretKeyRef:     mapSecretKeyRefAPIToKube(v.SecretKeyRef),
	}
}

func mapFieldRefAPIToKube(v *testkube.FieldRef) *corev1.ObjectFieldSelector {
	if v == nil {
		return nil
	}
	return &corev1.ObjectFieldSelector{
		APIVersion: v.ApiVersion,
		FieldPath:  v.FieldPath,
	}
}

func mapResourceFieldRefAPIToKube(v *testkube.ResourceFieldRef) *corev1.ResourceFieldSelector {
	if v == nil {
		return nil
	}
	divisor := resource.Quantity{}
	if v.Divisor != "" {
		if parsedDivisor, err := resource.ParseQuantity(v.Divisor); err == nil {
			divisor = parsedDivisor
		}
	}
	return &corev1.ResourceFieldSelector{
		ContainerName: v.ContainerName,
		Resource:      v.Resource,
		Divisor:       divisor,
	}
}

func mapConfigMapKeyRefAPIToKube(v *testkube.EnvVarSourceConfigMapKeyRef) *corev1.ConfigMapKeySelector {
	if v == nil {
		return nil
	}
	return &corev1.ConfigMapKeySelector{
		Key:                  v.Key,
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Optional:             v.Optional,
	}
}

func mapSecretKeyRefAPIToKube(v *testkube.EnvVarSourceSecretKeyRef) *corev1.SecretKeySelector {
	if v == nil {
		return nil
	}
	return &corev1.SecretKeySelector{
		Key:                  v.Key,
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Optional:             v.Optional,
	}
}

func mapGitAuthTypeAPIToKube(v *testkube.ContentGitAuthType) testsv3.GitAuthType {
	if v == nil {
		return ""
	}
	return testsv3.GitAuthType(*v)
}
