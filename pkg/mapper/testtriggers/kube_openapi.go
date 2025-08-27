package testtriggers

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
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
	var resource *testkube.TestTriggerResources
	if crd.Spec.Resource != "" {
		resource = (*testkube.TestTriggerResources)(&crd.Spec.Resource)
	}

	var action *testkube.TestTriggerActions
	if crd.Spec.Action != "" {
		action = (*testkube.TestTriggerActions)(&crd.Spec.Action)
	}

	var execution *testkube.TestTriggerExecutions
	if crd.Spec.Execution != "" {
		execution = (*testkube.TestTriggerExecutions)(&crd.Spec.Execution)
	}

	var concurrencyPolicy *testkube.TestTriggerConcurrencyPolicies
	if crd.Spec.ConcurrencyPolicy != "" {
		concurrencyPolicy = (*testkube.TestTriggerConcurrencyPolicies)(&crd.Spec.ConcurrencyPolicy)
	}

	return testkube.TestTrigger{
		Name:              crd.Name,
		Namespace:         crd.Namespace,
		Labels:            crd.Labels,
		Resource:          resource,
		ResourceSelector:  mapSelectorFromCRD(crd.Spec.ResourceSelector),
		Event:             string(crd.Spec.Event),
		ConditionSpec:     mapConditionSpecFromCRD(crd.Spec.ConditionSpec),
		ProbeSpec:         mapProbeSpecFromCRD(crd.Spec.ProbeSpec),
		ContentSelector:   mapContentSelectorFromCRD(crd.Spec.ContentSelector),
		Action:            action,
		ActionParameters:  mapActionParametersFromCRD(crd.Spec.ActionParameters),
		Execution:         execution,
		TestSelector:      mapSelectorFromCRD(crd.Spec.TestSelector),
		ConcurrencyPolicy: concurrencyPolicy,
		Disabled:          crd.Spec.Disabled,
	}
}

func mapSelectorFromCRD(selector testsv1.TestTriggerSelector) *testkube.TestTriggerSelector {
	var labelSelector *testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector
	if selector.LabelSelector != nil {
		labelSelector = mapLabelSelectorFromCRD(selector.LabelSelector)
	}
	return &testkube.TestTriggerSelector{
		Name:           selector.Name,
		NameRegex:      selector.NameRegex,
		Namespace:      selector.Namespace,
		NamespaceRegex: selector.NamespaceRegex,
		LabelSelector:  labelSelector,
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
		Delay:      conditionSpec.Delay,
		Conditions: conditions,
	}
}

func mapActionParametersFromCRD(actionParameters *testsv1.TestTriggerActionParameters) *testkube.TestTriggerActionParameters {
	if actionParameters == nil {
		return nil
	}

	return &testkube.TestTriggerActionParameters{
		Config: actionParameters.Config,
		Tags:   actionParameters.Tags,
		Target: common.MapPtr(actionParameters.Target, commonmapper.MapTargetKubeToAPI),
	}
}

func MapTestTriggerCRDToTestTriggerUpsertRequest(request testsv1.TestTrigger) testkube.TestTriggerUpsertRequest {
	var resource *testkube.TestTriggerResources
	if request.Spec.Resource != "" {
		resource = (*testkube.TestTriggerResources)(&request.Spec.Resource)
	}

	var action *testkube.TestTriggerActions
	if request.Spec.Action != "" {
		action = (*testkube.TestTriggerActions)(&request.Spec.Action)
	}

	var execution *testkube.TestTriggerExecutions
	if request.Spec.Execution != "" {
		execution = (*testkube.TestTriggerExecutions)(&request.Spec.Execution)
	}

	var concurrencyPolicy *testkube.TestTriggerConcurrencyPolicies
	if request.Spec.ConcurrencyPolicy != "" {
		concurrencyPolicy = (*testkube.TestTriggerConcurrencyPolicies)(&request.Spec.ConcurrencyPolicy)
	}

	return testkube.TestTriggerUpsertRequest{
		Name:              request.Name,
		Namespace:         request.Namespace,
		Labels:            request.Labels,
		Resource:          resource,
		ResourceSelector:  mapSelectorFromCRD(request.Spec.ResourceSelector),
		Event:             string(request.Spec.Event),
		ConditionSpec:     mapConditionSpecFromCRD(request.Spec.ConditionSpec),
		ProbeSpec:         mapProbeSpecFromCRD(request.Spec.ProbeSpec),
		ContentSelector:   mapContentSelectorFromCRD(request.Spec.ContentSelector),
		Action:            action,
		ActionParameters:  mapActionParametersFromCRD(request.Spec.ActionParameters),
		Execution:         execution,
		TestSelector:      mapSelectorFromCRD(request.Spec.TestSelector),
		ConcurrencyPolicy: concurrencyPolicy,
		Disabled:          request.Spec.Disabled,
	}
}

func mapProbeSpecFromCRD(probeSpec *testsv1.TestTriggerProbeSpec) *testkube.TestTriggerProbeSpec {
	if probeSpec == nil {
		return nil
	}

	var probes []testkube.TestTriggerProbe
	for _, probe := range probeSpec.Probes {
		var headers map[string]string
		if len(probe.Headers) != 0 {
			headers = make(map[string]string, len(probe.Headers))
			for key, value := range probe.Headers {
				headers[key] = value
			}
		}

		probes = append(probes, testkube.TestTriggerProbe{
			Scheme:  probe.Scheme,
			Host:    probe.Host,
			Path:    probe.Path,
			Port:    probe.Port,
			Headers: headers,
		})
	}

	return &testkube.TestTriggerProbeSpec{
		Timeout: probeSpec.Timeout,
		Delay:   probeSpec.Delay,
		Probes:  probes,
	}
}

func mapConfigMapKeyRefFromCRD(v *corev1.ConfigMapKeySelector) *testkube.EnvVarSourceConfigMapKeyRef {
	if v == nil {
		return nil
	}

	return &testkube.EnvVarSourceConfigMapKeyRef{
		Key:      v.Key,
		Name:     v.Name,
		Optional: v.Optional,
	}
}

func mapFieldRefFromCRD(v *corev1.ObjectFieldSelector) *testkube.FieldRef {
	if v == nil {
		return nil
	}

	return &testkube.FieldRef{
		ApiVersion: v.APIVersion,
		FieldPath:  v.FieldPath,
	}
}

func mapResourceFieldRefFromCRD(v *corev1.ResourceFieldSelector) *testkube.ResourceFieldRef {
	if v == nil {
		return nil
	}

	divisor := ""
	if !v.Divisor.IsZero() {
		divisor = v.Divisor.String()
	}

	return &testkube.ResourceFieldRef{
		ContainerName: v.ContainerName,
		Divisor:       divisor,
		Resource:      v.Resource,
	}
}

func mapSecretKeyRefFromCRD(v *corev1.SecretKeySelector) *testkube.EnvVarSourceSecretKeyRef {
	if v == nil {
		return nil
	}

	return &testkube.EnvVarSourceSecretKeyRef{
		Key:      v.Key,
		Name:     v.Name,
		Optional: v.Optional,
	}
}

func mapEnvVarSourceFromCRD(source *corev1.EnvVarSource) *testkube.EnvVarSource {
	if source == nil {
		return nil
	}

	return &testkube.EnvVarSource{
		ConfigMapKeyRef:  mapConfigMapKeyRefFromCRD(source.ConfigMapKeyRef),
		FieldRef:         mapFieldRefFromCRD(source.FieldRef),
		ResourceFieldRef: mapResourceFieldRefFromCRD(source.ResourceFieldRef),
		SecretKeyRef:     mapSecretKeyRefFromCRD(source.SecretKeyRef),
	}
}

func mapContentGitFromCRD(git *testsv1.TestTrggerContentGitSpec) *testkube.TestTriggerContentGit {
	if git == nil {
		return nil
	}

	return &testkube.TestTriggerContentGit{
		Uri:          git.Uri,
		Revision:     git.Revision,
		Username:     git.Username,
		UsernameFrom: mapEnvVarSourceFromCRD(git.UsernameFrom),
		Token:        git.Token,
		TokenFrom:    mapEnvVarSourceFromCRD(git.TokenFrom),
		SshKey:       git.SshKey,
		SshKeyFrom:   mapEnvVarSourceFromCRD(git.SshKeyFrom),
		AuthType:     (*testkube.ContentGitAuthType)(&git.AuthType),
		Paths:        git.Paths,
	}
}

func mapContentSelectorFromCRD(selector *testsv1.TestTrggerContentSelector) *testkube.TestTriggerContentSelector {
	if selector == nil {
		return nil
	}

	return &testkube.TestTriggerContentSelector{
		Git: mapContentGitFromCRD(selector.Git),
	}
}
