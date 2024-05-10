// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflows

import (
	"encoding/json"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapStringToIntOrString(i string) intstr.IntOrString {
	if v, err := strconv.ParseInt(i, 10, 32); err == nil {
		return intstr.IntOrString{Type: intstr.Int, IntVal: int32(v)}
	}
	return intstr.IntOrString{Type: intstr.String, StrVal: i}
}

func MapBoxedStringToIntOrString(v *testkube.BoxedString) *intstr.IntOrString {
	if v == nil {
		return nil
	}
	if vv, err := strconv.ParseInt(v.Value, 10, 32); err == nil {
		return &intstr.IntOrString{Type: intstr.Int, IntVal: int32(vv)}
	}
	return &intstr.IntOrString{Type: intstr.String, StrVal: v.Value}
}

func MapStringPtrToIntOrStringPtr(i *string) *intstr.IntOrString {
	if i == nil {
		return nil
	}
	return common.Ptr(MapStringToIntOrString(*i))
}

func MapBoxedStringToString(v *testkube.BoxedString) *string {
	if v == nil {
		return nil
	}
	return &v.Value
}

func MapBoxedStringToType[T ~string](v *testkube.BoxedString) *T {
	if v == nil {
		return nil
	}
	return common.Ptr(T(v.Value))
}

func MapBoxedStringToQuantity(v testkube.BoxedString) resource.Quantity {
	q, _ := resource.ParseQuantity(v.Value)
	return q
}

func MapBoxedBooleanToBool(v *testkube.BoxedBoolean) *bool {
	if v == nil {
		return nil
	}
	return &v.Value
}

func MapBoxedStringListToStringSlice(v *testkube.BoxedStringList) *[]string {
	if v == nil {
		return nil
	}
	return &v.Value
}

func MapBoxedIntegerToInt64(v *testkube.BoxedInteger) *int64 {
	if v == nil {
		return nil
	}
	return common.Ptr(int64(v.Value))
}

func MapBoxedIntegerToInt32(v *testkube.BoxedInteger) *int32 {
	if v == nil {
		return nil
	}
	return &v.Value
}

func MapDynamicListAPIToKube(v interface{}) *testworkflowsv1.DynamicList {
	var item testworkflowsv1.DynamicList
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	err = json.Unmarshal(b, &item)
	if err != nil {
		return nil
	}
	return &item
}

func MapDynamicListMapAPIToKube(v map[string]interface{}) map[string]testworkflowsv1.DynamicList {
	if len(v) == 0 {
		return nil
	}
	result := make(map[string]testworkflowsv1.DynamicList, len(v))
	for k := range v {
		item := MapDynamicListAPIToKube(v[k])
		if item != nil {
			result[k] = *item
		}
	}
	return result
}

func MapEnvVarAPIToKube(v testkube.EnvVar) corev1.EnvVar {
	return corev1.EnvVar{
		Name:      v.Name,
		Value:     v.Value,
		ValueFrom: common.MapPtr(v.ValueFrom, MapEnvVarSourceAPIToKube),
	}
}

func MapConfigMapKeyRefAPIToKube(v *testkube.EnvVarSourceConfigMapKeyRef) *corev1.ConfigMapKeySelector {
	if v == nil {
		return nil
	}
	return &corev1.ConfigMapKeySelector{
		Key:                  v.Key,
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Optional:             common.PtrOrNil(v.Optional),
	}
}

func MapFieldRefAPIToKube(v *testkube.EnvVarSourceFieldRef) *corev1.ObjectFieldSelector {
	if v == nil {
		return nil
	}
	return &corev1.ObjectFieldSelector{
		APIVersion: v.ApiVersion,
		FieldPath:  v.FieldPath,
	}
}

func MapResourceFieldRefAPIToKube(v *testkube.EnvVarSourceResourceFieldRef) *corev1.ResourceFieldSelector {
	if v == nil {
		return nil
	}
	divisor, _ := resource.ParseQuantity(v.Divisor)
	return &corev1.ResourceFieldSelector{
		ContainerName: v.ContainerName,
		Divisor:       divisor,
		Resource:      v.Resource,
	}
}

func MapSecretKeyRefAPIToKube(v *testkube.EnvVarSourceSecretKeyRef) *corev1.SecretKeySelector {
	if v == nil {
		return nil
	}
	return &corev1.SecretKeySelector{
		Key:                  v.Key,
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Optional:             common.PtrOrNil(v.Optional),
	}
}

func MapEnvVarSourceAPIToKube(v testkube.EnvVarSource) corev1.EnvVarSource {
	return corev1.EnvVarSource{
		ConfigMapKeyRef:  MapConfigMapKeyRefAPIToKube(v.ConfigMapKeyRef),
		FieldRef:         MapFieldRefAPIToKube(v.FieldRef),
		ResourceFieldRef: MapResourceFieldRefAPIToKube(v.ResourceFieldRef),
		SecretKeyRef:     MapSecretKeyRefAPIToKube(v.SecretKeyRef),
	}
}

func MapConfigMapEnvSourceAPIToKube(v *testkube.ConfigMapEnvSource) *corev1.ConfigMapEnvSource {
	if v == nil {
		return nil
	}
	return &corev1.ConfigMapEnvSource{
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Optional:             common.PtrOrNil(v.Optional),
	}
}

func MapSecretEnvSourceAPIToKube(v *testkube.SecretEnvSource) *corev1.SecretEnvSource {
	if v == nil {
		return nil
	}
	return &corev1.SecretEnvSource{
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Optional:             common.PtrOrNil(v.Optional),
	}
}

func MapEnvFromSourceAPIToKube(v testkube.EnvFromSource) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		Prefix:       v.Prefix,
		ConfigMapRef: MapConfigMapEnvSourceAPIToKube(v.ConfigMapRef),
		SecretRef:    MapSecretEnvSourceAPIToKube(v.SecretRef),
	}
}

func MapSecurityContextAPIToKube(v *testkube.SecurityContext) *corev1.SecurityContext {
	if v == nil {
		return nil
	}
	return &corev1.SecurityContext{
		Privileged:               MapBoxedBooleanToBool(v.Privileged),
		RunAsUser:                MapBoxedIntegerToInt64(v.RunAsUser),
		RunAsGroup:               MapBoxedIntegerToInt64(v.RunAsGroup),
		RunAsNonRoot:             MapBoxedBooleanToBool(v.RunAsNonRoot),
		ReadOnlyRootFilesystem:   MapBoxedBooleanToBool(v.ReadOnlyRootFilesystem),
		AllowPrivilegeEscalation: MapBoxedBooleanToBool(v.AllowPrivilegeEscalation),
	}
}

func MapLocalObjectReferenceAPIToKube(v testkube.LocalObjectReference) corev1.LocalObjectReference {
	return corev1.LocalObjectReference{Name: v.Name}
}

func MapConfigValueAPIToKube(v map[string]string) map[string]intstr.IntOrString {
	return common.MapMap(v, MapStringToIntOrString)
}

func MapParameterTypeAPIToKube(v *testkube.TestWorkflowParameterType) testworkflowsv1.ParameterType {
	if v == nil {
		return ""
	}
	return testworkflowsv1.ParameterType(*v)
}

func MapGitAuthTypeAPIToKube(v *testkube.ContentGitAuthType) testsv3.GitAuthType {
	if v == nil {
		return ""
	}
	return testsv3.GitAuthType(*v)
}

func MapImagePullPolicyAPIToKube(v *testkube.ImagePullPolicy) corev1.PullPolicy {
	if v == nil {
		return ""
	}
	return corev1.PullPolicy(*v)
}

func MapParameterSchemaAPIToKube(v testkube.TestWorkflowParameterSchema) testworkflowsv1.ParameterSchema {
	var example *intstr.IntOrString
	if v.Example != "" {
		example = common.Ptr(MapStringToIntOrString(v.Example))
	}
	return testworkflowsv1.ParameterSchema{
		Description: v.Description,
		Type:        MapParameterTypeAPIToKube(v.Type_),
		Enum:        v.Enum,
		Example:     example,
		Default:     MapStringPtrToIntOrStringPtr(MapBoxedStringToString(v.Default_)),
		ParameterStringSchema: testworkflowsv1.ParameterStringSchema{
			Format:    v.Format,
			Pattern:   v.Pattern,
			MinLength: MapBoxedIntegerToInt64(v.MinLength),
			MaxLength: MapBoxedIntegerToInt64(v.MaxLength),
		},
		ParameterNumberSchema: testworkflowsv1.ParameterNumberSchema{
			Minimum:          MapBoxedIntegerToInt64(v.Minimum),
			Maximum:          MapBoxedIntegerToInt64(v.Maximum),
			ExclusiveMinimum: MapBoxedIntegerToInt64(v.ExclusiveMinimum),
			ExclusiveMaximum: MapBoxedIntegerToInt64(v.ExclusiveMaximum),
			MultipleOf:       MapBoxedIntegerToInt64(v.MultipleOf),
		},
	}
}

func MapTemplateRefAPIToKube(v testkube.TestWorkflowTemplateRef) testworkflowsv1.TemplateRef {
	return testworkflowsv1.TemplateRef{
		Name:   v.Name,
		Config: MapConfigValueAPIToKube(v.Config),
	}
}

func MapContentGitAPIToKube(v testkube.TestWorkflowContentGit) testworkflowsv1.ContentGit {
	return testworkflowsv1.ContentGit{
		Uri:          v.Uri,
		Revision:     v.Revision,
		Username:     v.Username,
		UsernameFrom: common.MapPtr(v.UsernameFrom, MapEnvVarSourceAPIToKube),
		Token:        v.Token,
		TokenFrom:    common.MapPtr(v.TokenFrom, MapEnvVarSourceAPIToKube),
		AuthType:     MapGitAuthTypeAPIToKube(v.AuthType),
		MountPath:    v.MountPath,
		Paths:        v.Paths,
	}
}

func MapContentTarballAPIToKube(v testkube.TestWorkflowContentTarball) testworkflowsv1.ContentTarball {
	return testworkflowsv1.ContentTarball{
		Url:   v.Url,
		Path:  v.Path,
		Mount: MapBoxedBooleanToBool(v.Mount),
	}
}

func MapContentAPIToKube(v testkube.TestWorkflowContent) testworkflowsv1.Content {
	return testworkflowsv1.Content{
		Git:     common.MapPtr(v.Git, MapContentGitAPIToKube),
		Files:   common.MapSlice(v.Files, MapContentFileAPIToKube),
		Tarball: common.MapSlice(v.Tarball, MapContentTarballAPIToKube),
	}
}

func MapContentFileAPIToKube(v testkube.TestWorkflowContentFile) testworkflowsv1.ContentFile {
	return testworkflowsv1.ContentFile{
		Path:        v.Path,
		Content:     v.Content,
		ContentFrom: common.MapPtr(v.ContentFrom, MapEnvVarSourceAPIToKube),
		Mode:        MapBoxedIntegerToInt32(v.Mode),
	}
}

func MapResourcesListAPIToKube(v *testkube.TestWorkflowResourcesList) map[corev1.ResourceName]intstr.IntOrString {
	if v == nil {
		return nil
	}
	res := make(map[corev1.ResourceName]intstr.IntOrString)
	if v.Cpu != "" {
		res[corev1.ResourceCPU] = MapStringToIntOrString(v.Cpu)
	}
	if v.Memory != "" {
		res[corev1.ResourceMemory] = MapStringToIntOrString(v.Memory)
	}
	if v.Storage != "" {
		res[corev1.ResourceStorage] = MapStringToIntOrString(v.Storage)
	}
	if v.EphemeralStorage != "" {
		res[corev1.ResourceEphemeralStorage] = MapStringToIntOrString(v.EphemeralStorage)
	}
	return res
}

func MapResourcesAPIToKube(v testkube.TestWorkflowResources) testworkflowsv1.Resources {
	return testworkflowsv1.Resources{
		Limits:   MapResourcesListAPIToKube(v.Limits),
		Requests: MapResourcesListAPIToKube(v.Requests),
	}
}

func MapJobConfigAPIToKube(v testkube.TestWorkflowJobConfig) testworkflowsv1.JobConfig {
	return testworkflowsv1.JobConfig{
		Labels:                v.Labels,
		Annotations:           v.Annotations,
		Namespace:             v.Namespace,
		ActiveDeadlineSeconds: MapBoxedIntegerToInt64(v.ActiveDeadlineSeconds),
	}
}

func MapEventAPIToKube(v testkube.TestWorkflowEvent) testworkflowsv1.Event {
	return testworkflowsv1.Event{
		Cronjob: common.MapPtr(v.Cronjob, MapCronJobConfigAPIToKube),
	}
}

func MapCronJobConfigAPIToKube(v testkube.TestWorkflowCronJobConfig) testworkflowsv1.CronJobConfig {
	return testworkflowsv1.CronJobConfig{
		Cron:        v.Cron,
		Labels:      v.Labels,
		Annotations: v.Annotations,
	}
}
func MapHostPathVolumeSourceAPIToKube(v testkube.HostPathVolumeSource) corev1.HostPathVolumeSource {
	return corev1.HostPathVolumeSource{
		Path: v.Path,
		Type: MapBoxedStringToType[corev1.HostPathType](v.Type_),
	}
}

func MapEmptyDirVolumeSourceAPIToKube(v testkube.EmptyDirVolumeSource) corev1.EmptyDirVolumeSource {
	return corev1.EmptyDirVolumeSource{
		Medium:    corev1.StorageMedium(v.Medium),
		SizeLimit: common.MapPtr(v.SizeLimit, MapBoxedStringToQuantity),
	}
}

func MapGCEPersistentDiskVolumeSourceAPIToKube(v testkube.GcePersistentDiskVolumeSource) corev1.GCEPersistentDiskVolumeSource {
	return corev1.GCEPersistentDiskVolumeSource{
		PDName:    v.PdName,
		FSType:    v.FsType,
		Partition: v.Partition,
		ReadOnly:  v.ReadOnly,
	}
}

func MapAWSElasticBlockStoreVolumeSourceAPIToKube(v testkube.AwsElasticBlockStoreVolumeSource) corev1.AWSElasticBlockStoreVolumeSource {
	return corev1.AWSElasticBlockStoreVolumeSource{
		VolumeID:  v.VolumeID,
		FSType:    v.FsType,
		Partition: v.Partition,
		ReadOnly:  v.ReadOnly,
	}
}

func MapKeyToPathAPIToKube(v testkube.SecretVolumeSourceItems) corev1.KeyToPath {
	return corev1.KeyToPath{
		Key:  v.Key,
		Path: v.Path,
		Mode: MapBoxedIntegerToInt32(v.Mode),
	}
}

func MapSecretVolumeSourceAPIToKube(v testkube.SecretVolumeSource) corev1.SecretVolumeSource {
	return corev1.SecretVolumeSource{
		SecretName:  v.SecretName,
		Items:       common.MapSlice(v.Items, MapKeyToPathAPIToKube),
		DefaultMode: MapBoxedIntegerToInt32(v.DefaultMode),
		Optional:    common.PtrOrNil(v.Optional),
	}
}

func MapNFSVolumeSourceAPIToKube(v testkube.NfsVolumeSource) corev1.NFSVolumeSource {
	return corev1.NFSVolumeSource{
		Server:   v.Server,
		Path:     v.Path,
		ReadOnly: v.ReadOnly,
	}
}

func MapPersistentVolumeClaimVolumeSourceAPIToKube(v testkube.PersistentVolumeClaimVolumeSource) corev1.PersistentVolumeClaimVolumeSource {
	return corev1.PersistentVolumeClaimVolumeSource{
		ClaimName: v.ClaimName,
		ReadOnly:  v.ReadOnly,
	}
}

func MapCephFSVolumeSourceAPIToKube(v testkube.CephFsVolumeSource) corev1.CephFSVolumeSource {
	return corev1.CephFSVolumeSource{
		Monitors:   v.Monitors,
		Path:       v.Path,
		User:       v.User,
		SecretFile: v.SecretFile,
		SecretRef:  common.MapPtr(v.SecretRef, MapLocalObjectReferenceAPIToKube),
		ReadOnly:   v.ReadOnly,
	}
}

func MapAzureFileVolumeSourceAPIToKube(v testkube.AzureFileVolumeSource) corev1.AzureFileVolumeSource {
	return corev1.AzureFileVolumeSource{
		SecretName: v.SecretName,
		ShareName:  v.ShareName,
		ReadOnly:   v.ReadOnly,
	}
}

func MapConfigMapVolumeSourceAPIToKube(v testkube.ConfigMapVolumeSource) corev1.ConfigMapVolumeSource {
	return corev1.ConfigMapVolumeSource{
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Items:                common.MapSlice(v.Items, MapKeyToPathAPIToKube),
		DefaultMode:          MapBoxedIntegerToInt32(v.DefaultMode),
		Optional:             common.PtrOrNil(v.Optional),
	}
}

func MapAzureDiskVolumeSourceAPIToKube(v testkube.AzureDiskVolumeSource) corev1.AzureDiskVolumeSource {
	return corev1.AzureDiskVolumeSource{
		DiskName:    v.DiskName,
		DataDiskURI: v.DiskURI,
		CachingMode: MapBoxedStringToType[corev1.AzureDataDiskCachingMode](v.CachingMode),
		FSType:      MapBoxedStringToString(v.FsType),
		ReadOnly:    common.PtrOrNil(v.ReadOnly),
		Kind:        MapBoxedStringToType[corev1.AzureDataDiskKind](v.Kind),
	}
}

func MapVolumeAPIToKube(v testkube.Volume) corev1.Volume {
	// TODO: Add rest of VolumeSource types in future,
	//       so they will be recognized by JSON API and persisted with Execution.
	return corev1.Volume{
		Name: v.Name,
		VolumeSource: corev1.VolumeSource{
			HostPath:              common.MapPtr(v.HostPath, MapHostPathVolumeSourceAPIToKube),
			EmptyDir:              common.MapPtr(v.EmptyDir, MapEmptyDirVolumeSourceAPIToKube),
			GCEPersistentDisk:     common.MapPtr(v.GcePersistentDisk, MapGCEPersistentDiskVolumeSourceAPIToKube),
			AWSElasticBlockStore:  common.MapPtr(v.AwsElasticBlockStore, MapAWSElasticBlockStoreVolumeSourceAPIToKube),
			Secret:                common.MapPtr(v.Secret, MapSecretVolumeSourceAPIToKube),
			NFS:                   common.MapPtr(v.Nfs, MapNFSVolumeSourceAPIToKube),
			PersistentVolumeClaim: common.MapPtr(v.PersistentVolumeClaim, MapPersistentVolumeClaimVolumeSourceAPIToKube),
			CephFS:                common.MapPtr(v.Cephfs, MapCephFSVolumeSourceAPIToKube),
			AzureFile:             common.MapPtr(v.AzureFile, MapAzureFileVolumeSourceAPIToKube),
			ConfigMap:             common.MapPtr(v.ConfigMap, MapConfigMapVolumeSourceAPIToKube),
			AzureDisk:             common.MapPtr(v.AzureDisk, MapAzureDiskVolumeSourceAPIToKube),
		},
	}
}

func MapTolerationAPIToKube(v testkube.Toleration) corev1.Toleration {
	return corev1.Toleration{
		Key:               v.Key,
		Operator:          corev1.TolerationOperator(v.Operator),
		Value:             v.Value,
		Effect:            corev1.TaintEffect(v.Effect),
		TolerationSeconds: MapBoxedIntegerToInt64(v.TolerationSeconds),
	}
}

func MapHostAliasAPIToKube(v testkube.HostAlias) corev1.HostAlias {
	return corev1.HostAlias{IP: v.Ip, Hostnames: v.Hostnames}
}

func MapTopologySpreadConstraintAPIToKube(v testkube.TopologySpreadConstraint) corev1.TopologySpreadConstraint {
	return corev1.TopologySpreadConstraint{
		MaxSkew:            v.MaxSkew,
		TopologyKey:        v.TopologyKey,
		WhenUnsatisfiable:  corev1.UnsatisfiableConstraintAction(v.WhenUnsatisfiable),
		LabelSelector:      common.MapPtr(v.LabelSelector, MapLabelSelectorAPIToKube),
		MinDomains:         MapBoxedIntegerToInt32(v.MinDomains),
		NodeAffinityPolicy: common.MapPtr(MapBoxedStringToString(v.NodeAffinityPolicy), common.MapStringToEnum[corev1.NodeInclusionPolicy]),
		NodeTaintsPolicy:   common.MapPtr(MapBoxedStringToString(v.NodeTaintsPolicy), common.MapStringToEnum[corev1.NodeInclusionPolicy]),
		MatchLabelKeys:     v.MatchLabelKeys,
	}
}

func MapPodSchedulingGateAPIToKube(v testkube.PodSchedulingGate) corev1.PodSchedulingGate {
	return corev1.PodSchedulingGate{Name: v.Name}
}

func MapPodResourceClaimAPIToKube(v testkube.PodResourceClaim) corev1.PodResourceClaim {
	source := testkube.ClaimSource{}
	if v.Source != nil {
		source = *v.Source
	}
	return corev1.PodResourceClaim{
		Name: v.Name,
		Source: corev1.ClaimSource{
			ResourceClaimName:         MapBoxedStringToString(source.ResourceClaimName),
			ResourceClaimTemplateName: MapBoxedStringToString(source.ResourceClaimTemplateName),
		},
	}
}

func MapPodSecurityContextAPIToKube(v testkube.PodSecurityContext) corev1.PodSecurityContext {
	return corev1.PodSecurityContext{
		RunAsUser:    MapBoxedIntegerToInt64(v.RunAsUser),
		RunAsGroup:   MapBoxedIntegerToInt64(v.RunAsGroup),
		RunAsNonRoot: MapBoxedBooleanToBool(v.RunAsNonRoot),
	}
}

func MapNodeSelectorRequirementAPIToKube(v testkube.NodeSelectorRequirement) corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      v.Key,
		Operator: corev1.NodeSelectorOperator(v.Operator),
		Values:   v.Values,
	}
}

func MapNodeSelectorTermAPIToKube(v testkube.NodeSelectorTerm) corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: common.MapSlice(v.MatchExpressions, MapNodeSelectorRequirementAPIToKube),
		MatchFields:      common.MapSlice(v.MatchFields, MapNodeSelectorRequirementAPIToKube),
	}
}

func MapNodeSelectorAPIToKube(v testkube.NodeSelector) corev1.NodeSelector {
	return corev1.NodeSelector{
		NodeSelectorTerms: common.MapSlice(v.NodeSelectorTerms, MapNodeSelectorTermAPIToKube),
	}
}

func MapPreferredSchedulingTermAPIToKube(v testkube.PreferredSchedulingTerm) corev1.PreferredSchedulingTerm {
	return corev1.PreferredSchedulingTerm{
		Weight:     v.Weight,
		Preference: MapNodeSelectorTermAPIToKube(common.ResolvePtr(v.Preference, testkube.NodeSelectorTerm{})),
	}
}

func MapNodeAffinityAPIToKube(v testkube.NodeAffinity) corev1.NodeAffinity {
	return corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution:  common.MapPtr(v.RequiredDuringSchedulingIgnoredDuringExecution, MapNodeSelectorAPIToKube),
		PreferredDuringSchedulingIgnoredDuringExecution: common.MapSlice(v.PreferredDuringSchedulingIgnoredDuringExecution, MapPreferredSchedulingTermAPIToKube),
	}
}

func MapPodAffinityAPIToKube(v testkube.PodAffinity) corev1.PodAffinity {
	return corev1.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution:  common.MapSlice(v.RequiredDuringSchedulingIgnoredDuringExecution, MapPodAffinityTermAPIToKube),
		PreferredDuringSchedulingIgnoredDuringExecution: common.MapSlice(v.PreferredDuringSchedulingIgnoredDuringExecution, MapWeightedPodAffinityTermAPIToKube),
	}
}

func MapLabelSelectorRequirementAPIToKube(v testkube.LabelSelectorRequirement) metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      v.Key,
		Operator: metav1.LabelSelectorOperator(v.Operator),
		Values:   v.Values,
	}
}

func MapLabelSelectorAPIToKube(v testkube.LabelSelector) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels:      v.MatchLabels,
		MatchExpressions: common.MapSlice(v.MatchExpressions, MapLabelSelectorRequirementAPIToKube),
	}
}

func MapPodAffinityTermAPIToKube(v testkube.PodAffinityTerm) corev1.PodAffinityTerm {
	return corev1.PodAffinityTerm{
		LabelSelector:     common.MapPtr(v.LabelSelector, MapLabelSelectorAPIToKube),
		Namespaces:        v.Namespaces,
		TopologyKey:       v.TopologyKey,
		NamespaceSelector: common.MapPtr(v.NamespaceSelector, MapLabelSelectorAPIToKube),
	}
}

func MapWeightedPodAffinityTermAPIToKube(v testkube.WeightedPodAffinityTerm) corev1.WeightedPodAffinityTerm {
	return corev1.WeightedPodAffinityTerm{
		Weight:          v.Weight,
		PodAffinityTerm: MapPodAffinityTermAPIToKube(common.ResolvePtr(v.PodAffinityTerm, testkube.PodAffinityTerm{})),
	}
}

func MapPodAntiAffinityAPIToKube(v testkube.PodAffinity) corev1.PodAntiAffinity {
	return corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution:  common.MapSlice(v.RequiredDuringSchedulingIgnoredDuringExecution, MapPodAffinityTermAPIToKube),
		PreferredDuringSchedulingIgnoredDuringExecution: common.MapSlice(v.PreferredDuringSchedulingIgnoredDuringExecution, MapWeightedPodAffinityTermAPIToKube),
	}
}

func MapAffinityAPIToKube(v testkube.Affinity) corev1.Affinity {
	return corev1.Affinity{
		NodeAffinity:    common.MapPtr(v.NodeAffinity, MapNodeAffinityAPIToKube),
		PodAffinity:     common.MapPtr(v.PodAffinity, MapPodAffinityAPIToKube),
		PodAntiAffinity: common.MapPtr(v.PodAntiAffinity, MapPodAntiAffinityAPIToKube),
	}
}

func MapPodDNSConfigOptionAPIToKube(v testkube.PodDnsConfigOption) corev1.PodDNSConfigOption {
	return corev1.PodDNSConfigOption{
		Name:  v.Name,
		Value: MapBoxedStringToString(v.Value),
	}
}

func MapPodDNSConfigAPIToKube(v testkube.PodDnsConfig) corev1.PodDNSConfig {
	return corev1.PodDNSConfig{
		Nameservers: v.Nameservers,
		Searches:    v.Searches,
		Options:     common.MapSlice(v.Options, MapPodDNSConfigOptionAPIToKube),
	}
}

func MapPodConfigAPIToKube(v testkube.TestWorkflowPodConfig) testworkflowsv1.PodConfig {
	return testworkflowsv1.PodConfig{
		ServiceAccountName:        v.ServiceAccountName,
		ImagePullSecrets:          common.MapSlice(v.ImagePullSecrets, MapLocalObjectReferenceAPIToKube),
		NodeSelector:              v.NodeSelector,
		Labels:                    v.Labels,
		Annotations:               v.Annotations,
		Volumes:                   common.MapSlice(v.Volumes, MapVolumeAPIToKube),
		ActiveDeadlineSeconds:     MapBoxedIntegerToInt64(v.ActiveDeadlineSeconds),
		DNSPolicy:                 corev1.DNSPolicy(v.DnsPolicy),
		NodeName:                  v.NodeName,
		SecurityContext:           common.MapPtr(v.SecurityContext, MapPodSecurityContextAPIToKube),
		Hostname:                  v.Hostname,
		Subdomain:                 v.Subdomain,
		Affinity:                  common.MapPtr(v.Affinity, MapAffinityAPIToKube),
		Tolerations:               common.MapSlice(v.Tolerations, MapTolerationAPIToKube),
		HostAliases:               common.MapSlice(v.HostAliases, MapHostAliasAPIToKube),
		PriorityClassName:         v.PriorityClassName,
		Priority:                  MapBoxedIntegerToInt32(v.Priority),
		DNSConfig:                 common.MapPtr(v.DnsConfig, MapPodDNSConfigAPIToKube),
		PreemptionPolicy:          common.MapPtr(MapBoxedStringToString(v.PreemptionPolicy), common.MapStringToEnum[corev1.PreemptionPolicy]),
		TopologySpreadConstraints: common.MapSlice(v.TopologySpreadConstraints, MapTopologySpreadConstraintAPIToKube),
		SchedulingGates:           common.MapSlice(v.SchedulingGates, MapPodSchedulingGateAPIToKube),
		ResourceClaims:            common.MapSlice(v.ResourceClaims, MapPodResourceClaimAPIToKube),
	}
}

func MapVolumeMountAPIToKube(v testkube.VolumeMount) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:             v.Name,
		ReadOnly:         v.ReadOnly,
		MountPath:        v.MountPath,
		SubPath:          v.SubPath,
		MountPropagation: MapBoxedStringToType[corev1.MountPropagationMode](v.MountPropagation),
		SubPathExpr:      v.SubPathExpr,
	}
}

func MapContainerConfigAPIToKube(v testkube.TestWorkflowContainerConfig) testworkflowsv1.ContainerConfig {
	return testworkflowsv1.ContainerConfig{
		WorkingDir:      MapBoxedStringToString(v.WorkingDir),
		Image:           v.Image,
		ImagePullPolicy: MapImagePullPolicyAPIToKube(v.ImagePullPolicy),
		Env:             common.MapSlice(v.Env, MapEnvVarAPIToKube),
		EnvFrom:         common.MapSlice(v.EnvFrom, MapEnvFromSourceAPIToKube),
		Command:         MapBoxedStringListToStringSlice(v.Command),
		Args:            MapBoxedStringListToStringSlice(v.Args),
		Resources:       common.MapPtr(v.Resources, MapResourcesAPIToKube),
		SecurityContext: MapSecurityContextAPIToKube(v.SecurityContext),
		VolumeMounts:    common.MapSlice(v.VolumeMounts, MapVolumeMountAPIToKube),
	}
}

func MapStepRunAPIToKube(v testkube.TestWorkflowStepRun) testworkflowsv1.StepRun {
	return testworkflowsv1.StepRun{
		ContainerConfig: testworkflowsv1.ContainerConfig{
			WorkingDir:      MapBoxedStringToString(v.WorkingDir),
			Image:           v.Image,
			ImagePullPolicy: MapImagePullPolicyAPIToKube(v.ImagePullPolicy),
			Env:             common.MapSlice(v.Env, MapEnvVarAPIToKube),
			EnvFrom:         common.MapSlice(v.EnvFrom, MapEnvFromSourceAPIToKube),
			Command:         MapBoxedStringListToStringSlice(v.Command),
			Args:            MapBoxedStringListToStringSlice(v.Args),
			Resources:       common.MapPtr(v.Resources, MapResourcesAPIToKube),
			SecurityContext: MapSecurityContextAPIToKube(v.SecurityContext),
			VolumeMounts:    common.MapSlice(v.VolumeMounts, MapVolumeMountAPIToKube),
		},
		Shell: MapBoxedStringToString(v.Shell),
	}
}

func MapTestVariableAPIToKube(v testkube.Variable) testsv3.Variable {
	var valueFrom corev1.EnvVarSource
	if v.ConfigMapRef != nil {
		valueFrom.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: v.ConfigMapRef.Name},
			Key:                  v.ConfigMapRef.Key,
		}
	}
	if v.SecretRef != nil {
		valueFrom.SecretKeyRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: v.SecretRef.Name},
			Key:                  v.SecretRef.Key,
		}
	}
	return testsv3.Variable{
		Type_:     string(common.ResolvePtr[testkube.VariableType](v.Type_, "")),
		Name:      v.Name,
		Value:     v.Value,
		ValueFrom: valueFrom,
	}
}

func MapTestArtifactRequestAPIToKube(v testkube.ArtifactRequest) testsv3.ArtifactRequest {
	return testsv3.ArtifactRequest{
		StorageClassName:           v.StorageClassName,
		VolumeMountPath:            v.VolumeMountPath,
		Dirs:                       v.Dirs,
		Masks:                      v.Masks,
		StorageBucket:              v.StorageBucket,
		OmitFolderPerExecution:     v.OmitFolderPerExecution,
		SharedBetweenPods:          v.SharedBetweenPods,
		UseDefaultStorageClassName: v.UseDefaultStorageClassName,
	}
}

func MapTestEnvReferenceAPIToKube(v testkube.EnvReference) testsv3.EnvReference {
	return testsv3.EnvReference{
		LocalObjectReference: common.ResolvePtr(common.MapPtr(v.Reference, MapLocalObjectReferenceAPIToKube), corev1.LocalObjectReference{}),
		Mount:                v.Mount,
		MountPath:            v.MountPath,
		MapToVariables:       v.MapToVariables,
	}
}

func MapStepExecuteTestExecutionRequestAPIToKube(v testkube.TestWorkflowStepExecuteTestExecutionRequest) testworkflowsv1.TestExecutionRequest {
	return testworkflowsv1.TestExecutionRequest{
		Name:                               v.Name,
		ExecutionLabels:                    v.ExecutionLabels,
		VariablesFile:                      v.VariablesFile,
		IsVariablesFileUploaded:            v.IsVariablesFileUploaded,
		Variables:                          common.MapMap(v.Variables, MapTestVariableAPIToKube),
		TestSecretUUID:                     v.TestSecretUUID,
		Args:                               v.Args,
		ArgsMode:                           testsv3.ArgsModeType(v.ArgsMode),
		Command:                            v.Command,
		Image:                              v.Image,
		ImagePullSecrets:                   common.MapSlice(v.ImagePullSecrets, MapLocalObjectReferenceAPIToKube),
		Sync:                               v.Sync,
		HttpProxy:                          v.HttpProxy,
		HttpsProxy:                         v.HttpsProxy,
		NegativeTest:                       v.NegativeTest,
		ActiveDeadlineSeconds:              v.ActiveDeadlineSeconds,
		ArtifactRequest:                    common.MapPtr(v.ArtifactRequest, MapTestArtifactRequestAPIToKube),
		JobTemplate:                        v.JobTemplate,
		CronJobTemplate:                    v.CronJobTemplate,
		PreRunScript:                       v.PreRunScript,
		PostRunScript:                      v.PostRunScript,
		ExecutePostRunScriptBeforeScraping: v.ExecutePostRunScriptBeforeScraping,
		SourceScripts:                      v.SourceScripts,
		ScraperTemplate:                    v.ScraperTemplate,
		EnvConfigMaps:                      common.MapSlice(v.EnvConfigMaps, MapTestEnvReferenceAPIToKube),
		EnvSecrets:                         common.MapSlice(v.EnvSecrets, MapTestEnvReferenceAPIToKube),
		ExecutionNamespace:                 v.ExecutionNamespace,
	}
}

func MapTarballFilePatternAPIToKube(v testkube.TestWorkflowTarballFilePattern) *testworkflowsv1.DynamicList {
	if v.Expression != "" {
		return MapDynamicListAPIToKube(v.Expression)
	}
	return MapDynamicListAPIToKube(v.Static)
}

func MapTarballRequestAPIToKube(v testkube.TestWorkflowTarballRequest) testworkflowsv1.TarballRequest {
	var files *testworkflowsv1.DynamicList
	if v.Files != nil {
		files = MapTarballFilePatternAPIToKube(*v.Files)
	}
	return testworkflowsv1.TarballRequest{
		From:  v.From,
		Files: files,
	}
}

func MapStepExecuteTestAPIToKube(v testkube.TestWorkflowStepExecuteTestRef) testworkflowsv1.StepExecuteTest {
	return testworkflowsv1.StepExecuteTest{
		Name:             v.Name,
		Description:      v.Description,
		ExecutionRequest: common.MapPtr(v.ExecutionRequest, MapStepExecuteTestExecutionRequestAPIToKube),
		Tarball:          common.MapMap(v.Tarball, MapTarballRequestAPIToKube),
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count:    MapBoxedStringToIntOrString(v.Count),
			MaxCount: MapBoxedStringToIntOrString(v.MaxCount),
			Matrix:   MapDynamicListMapAPIToKube(v.Matrix),
			Shards:   MapDynamicListMapAPIToKube(v.Shards),
		},
	}
}

func MapStepExecuteTestWorkflowAPIToKube(v testkube.TestWorkflowStepExecuteTestWorkflowRef) testworkflowsv1.StepExecuteWorkflow {
	return testworkflowsv1.StepExecuteWorkflow{
		Name:          v.Name,
		Description:   v.Description,
		ExecutionName: v.ExecutionName,
		Tarball:       common.MapMap(v.Tarball, MapTarballRequestAPIToKube),
		Config:        MapConfigValueAPIToKube(v.Config),
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count:    MapBoxedStringToIntOrString(v.Count),
			MaxCount: MapBoxedStringToIntOrString(v.MaxCount),
			Matrix:   MapDynamicListMapAPIToKube(v.Matrix),
			Shards:   MapDynamicListMapAPIToKube(v.Shards),
		},
	}
}

func MapStepExecuteAPIToKube(v testkube.TestWorkflowStepExecute) testworkflowsv1.StepExecute {
	return testworkflowsv1.StepExecute{
		Parallelism: v.Parallelism,
		Async:       v.Async,
		Tests:       common.MapSlice(v.Tests, MapStepExecuteTestAPIToKube),
		Workflows:   common.MapSlice(v.Workflows, MapStepExecuteTestWorkflowAPIToKube),
	}
}

func MapStepArtifactsCompressionAPIToKube(v testkube.TestWorkflowStepArtifactsCompression) testworkflowsv1.ArtifactCompression {
	return testworkflowsv1.ArtifactCompression{
		Name: v.Name,
	}
}

func MapStepArtifactsAPIToKube(v testkube.TestWorkflowStepArtifacts) testworkflowsv1.StepArtifacts {
	return testworkflowsv1.StepArtifacts{
		WorkingDir: MapBoxedStringToString(v.WorkingDir),
		Compress:   common.MapPtr(v.Compress, MapStepArtifactsCompressionAPIToKube),
		Paths:      v.Paths,
	}
}

func MapRetryPolicyAPIToKube(v testkube.TestWorkflowRetryPolicy) testworkflowsv1.RetryPolicy {
	return testworkflowsv1.RetryPolicy{
		Count: v.Count,
		Until: v.Until,
	}
}

func MapStepParallelTransferAPIToKube(v testkube.TestWorkflowStepParallelTransfer) testworkflowsv1.StepParallelTransfer {
	return testworkflowsv1.StepParallelTransfer{
		From:  v.From,
		To:    v.To,
		Files: common.ResolvePtr(common.MapPtr(v.Files, MapTarballFilePatternAPIToKube), nil),
		Mount: MapBoxedBooleanToBool(v.Mount),
	}
}

func MapStepParallelAPIToKube(v testkube.TestWorkflowStepParallel) testworkflowsv1.StepParallel {
	return testworkflowsv1.StepParallel{
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count:    MapBoxedStringToIntOrString(v.Count),
			MaxCount: MapBoxedStringToIntOrString(v.MaxCount),
			Matrix:   MapDynamicListMapAPIToKube(v.Matrix),
			Shards:   MapDynamicListMapAPIToKube(v.Shards),
		},
		Transfer: nil,
		TestWorkflowSpec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Config:    common.MapMap(v.Config, MapParameterSchemaAPIToKube),
				Content:   common.MapPtr(v.Content, MapContentAPIToKube),
				Container: common.MapPtr(v.Container, MapContainerConfigAPIToKube),
				Job:       common.MapPtr(v.Job, MapJobConfigAPIToKube),
				Pod:       common.MapPtr(v.Pod, MapPodConfigAPIToKube),
				Events:    common.MapSlice(v.Events, MapEventAPIToKube),
			},
			Use:   common.MapSlice(v.Use, MapTemplateRefAPIToKube),
			Setup: common.MapSlice(v.Setup, MapStepAPIToKube),
			Steps: common.MapSlice(v.Steps, MapStepAPIToKube),
			After: common.MapSlice(v.After, MapStepAPIToKube),
		},
	}
}

func MapIndependentStepParallelAPIToKube(v testkube.TestWorkflowIndependentStepParallel) testworkflowsv1.IndependentStepParallel {
	return testworkflowsv1.IndependentStepParallel{
		StepExecuteStrategy: testworkflowsv1.StepExecuteStrategy{
			Count:    MapBoxedStringToIntOrString(v.Count),
			MaxCount: MapBoxedStringToIntOrString(v.MaxCount),
			Matrix:   MapDynamicListMapAPIToKube(v.Matrix),
			Shards:   MapDynamicListMapAPIToKube(v.Shards),
		},
		Transfer: nil,
		TestWorkflowTemplateSpec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Config:    common.MapMap(v.Config, MapParameterSchemaAPIToKube),
				Content:   common.MapPtr(v.Content, MapContentAPIToKube),
				Container: common.MapPtr(v.Container, MapContainerConfigAPIToKube),
				Job:       common.MapPtr(v.Job, MapJobConfigAPIToKube),
				Pod:       common.MapPtr(v.Pod, MapPodConfigAPIToKube),
				Events:    common.MapSlice(v.Events, MapEventAPIToKube),
			},
			Setup: common.MapSlice(v.Setup, MapIndependentStepAPIToKube),
			Steps: common.MapSlice(v.Steps, MapIndependentStepAPIToKube),
			After: common.MapSlice(v.After, MapIndependentStepAPIToKube),
		},
	}
}

func MapStepAPIToKube(v testkube.TestWorkflowStep) testworkflowsv1.Step {
	return testworkflowsv1.Step{
		StepBase: testworkflowsv1.StepBase{
			Name:       v.Name,
			Condition:  v.Condition,
			Paused:     v.Paused,
			Negative:   v.Negative,
			Optional:   v.Optional,
			Retry:      common.MapPtr(v.Retry, MapRetryPolicyAPIToKube),
			Timeout:    v.Timeout,
			Delay:      v.Delay,
			Content:    common.MapPtr(v.Content, MapContentAPIToKube),
			Shell:      v.Shell,
			Run:        common.MapPtr(v.Run, MapStepRunAPIToKube),
			WorkingDir: MapBoxedStringToString(v.WorkingDir),
			Container:  common.MapPtr(v.Container, MapContainerConfigAPIToKube),
			Execute:    common.MapPtr(v.Execute, MapStepExecuteAPIToKube),
			Artifacts:  common.MapPtr(v.Artifacts, MapStepArtifactsAPIToKube),
		},
		Use:      common.MapSlice(v.Use, MapTemplateRefAPIToKube),
		Template: common.MapPtr(v.Template, MapTemplateRefAPIToKube),
		Setup:    common.MapSlice(v.Setup, MapStepAPIToKube),
		Steps:    common.MapSlice(v.Steps, MapStepAPIToKube),
		Parallel: common.MapPtr(v.Parallel, MapStepParallelAPIToKube),
	}
}

func MapIndependentStepAPIToKube(v testkube.TestWorkflowIndependentStep) testworkflowsv1.IndependentStep {
	return testworkflowsv1.IndependentStep{
		StepBase: testworkflowsv1.StepBase{
			Name:       v.Name,
			Condition:  v.Condition,
			Paused:     v.Paused,
			Negative:   v.Negative,
			Optional:   v.Optional,
			Retry:      common.MapPtr(v.Retry, MapRetryPolicyAPIToKube),
			Timeout:    v.Timeout,
			Delay:      v.Delay,
			Content:    common.MapPtr(v.Content, MapContentAPIToKube),
			Shell:      v.Shell,
			Run:        common.MapPtr(v.Run, MapStepRunAPIToKube),
			WorkingDir: MapBoxedStringToString(v.WorkingDir),
			Container:  common.MapPtr(v.Container, MapContainerConfigAPIToKube),
			Execute:    common.MapPtr(v.Execute, MapStepExecuteAPIToKube),
			Artifacts:  common.MapPtr(v.Artifacts, MapStepArtifactsAPIToKube),
		},
		Setup:    common.MapSlice(v.Setup, MapIndependentStepAPIToKube),
		Steps:    common.MapSlice(v.Steps, MapIndependentStepAPIToKube),
		Parallel: common.MapPtr(v.Parallel, MapIndependentStepParallelAPIToKube),
	}
}

func MapSpecAPIToKube(v testkube.TestWorkflowSpec) testworkflowsv1.TestWorkflowSpec {
	return testworkflowsv1.TestWorkflowSpec{
		TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
			Config:    common.MapMap(v.Config, MapParameterSchemaAPIToKube),
			Content:   common.MapPtr(v.Content, MapContentAPIToKube),
			Container: common.MapPtr(v.Container, MapContainerConfigAPIToKube),
			Job:       common.MapPtr(v.Job, MapJobConfigAPIToKube),
			Pod:       common.MapPtr(v.Pod, MapPodConfigAPIToKube),
			Events:    common.MapSlice(v.Events, MapEventAPIToKube),
		},
		Use:   common.MapSlice(v.Use, MapTemplateRefAPIToKube),
		Setup: common.MapSlice(v.Setup, MapStepAPIToKube),
		Steps: common.MapSlice(v.Steps, MapStepAPIToKube),
		After: common.MapSlice(v.After, MapStepAPIToKube),
	}
}

func MapTemplateSpecAPIToKube(v testkube.TestWorkflowTemplateSpec) testworkflowsv1.TestWorkflowTemplateSpec {
	return testworkflowsv1.TestWorkflowTemplateSpec{
		TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
			Config:    common.MapMap(v.Config, MapParameterSchemaAPIToKube),
			Content:   common.MapPtr(v.Content, MapContentAPIToKube),
			Container: common.MapPtr(v.Container, MapContainerConfigAPIToKube),
			Job:       common.MapPtr(v.Job, MapJobConfigAPIToKube),
			Pod:       common.MapPtr(v.Pod, MapPodConfigAPIToKube),
			Events:    common.MapSlice(v.Events, MapEventAPIToKube),
		},
		Setup: common.MapSlice(v.Setup, MapIndependentStepAPIToKube),
		Steps: common.MapSlice(v.Steps, MapIndependentStepAPIToKube),
		After: common.MapSlice(v.After, MapIndependentStepAPIToKube),
	}
}

func MapTestWorkflowAPIToKube(w testkube.TestWorkflow) testworkflowsv1.TestWorkflow {
	return testworkflowsv1.TestWorkflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestWorkflow",
			APIVersion: testworkflowsv1.GroupVersion.Group + "/" + testworkflowsv1.GroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              w.Name,
			Namespace:         w.Namespace,
			Labels:            w.Labels,
			Annotations:       w.Annotations,
			CreationTimestamp: metav1.Time{Time: w.Created},
		},
		Description: w.Description,
		Spec:        common.ResolvePtr(common.MapPtr(w.Spec, MapSpecAPIToKube), testworkflowsv1.TestWorkflowSpec{}),
	}
}

func MapTestWorkflowTemplateAPIToKube(w testkube.TestWorkflowTemplate) testworkflowsv1.TestWorkflowTemplate {
	return testworkflowsv1.TestWorkflowTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestWorkflowTemplate",
			APIVersion: testworkflowsv1.GroupVersion.Group + "/" + testworkflowsv1.GroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              w.Name,
			Namespace:         w.Namespace,
			Labels:            w.Labels,
			Annotations:       w.Annotations,
			CreationTimestamp: metav1.Time{Time: w.Created},
		},
		Description: w.Description,
		Spec:        common.ResolvePtr(common.MapPtr(w.Spec, MapTemplateSpecAPIToKube), testworkflowsv1.TestWorkflowTemplateSpec{}),
	}
}

func MapTemplateAPIToKube(w *testkube.TestWorkflowTemplate) *testworkflowsv1.TestWorkflowTemplate {
	return common.MapPtr(w, MapTestWorkflowTemplateAPIToKube)
}

func MapAPIToKube(w *testkube.TestWorkflow) *testworkflowsv1.TestWorkflow {
	return common.MapPtr(w, MapTestWorkflowAPIToKube)
}

func MapListAPIToKube(v []testkube.TestWorkflow) testworkflowsv1.TestWorkflowList {
	items := make([]testworkflowsv1.TestWorkflow, len(v))
	for i, item := range v {
		items[i] = MapTestWorkflowAPIToKube(item)
	}
	return testworkflowsv1.TestWorkflowList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestWorkflowList",
			APIVersion: testworkflowsv1.GroupVersion.String(),
		},
		Items: items,
	}
}

func MapTemplateListAPIToKube(v []testkube.TestWorkflowTemplate) testworkflowsv1.TestWorkflowTemplateList {
	items := make([]testworkflowsv1.TestWorkflowTemplate, len(v))
	for i, item := range v {
		items[i] = MapTestWorkflowTemplateAPIToKube(item)
	}
	return testworkflowsv1.TestWorkflowTemplateList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestWorkflowTemplateList",
			APIVersion: testworkflowsv1.GroupVersion.String(),
		},
		Items: items,
	}
}
