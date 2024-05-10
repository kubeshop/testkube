// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflows

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapIntOrStringToString(i intstr.IntOrString) string {
	return i.String()
}

func MapIntOrStringPtrToStringPtr(i *intstr.IntOrString) *string {
	if i == nil {
		return nil
	}
	return common.Ptr(MapIntOrStringToString(*i))
}

func MapIntOrStringToBoxedString(v *intstr.IntOrString) *testkube.BoxedString {
	if v == nil {
		return nil
	}
	return MapStringToBoxedString(common.Ptr(v.String()))
}

func MapStringToBoxedString(v *string) *testkube.BoxedString {
	if v == nil {
		return nil
	}
	return &testkube.BoxedString{Value: *v}
}

func MapStringTypeToBoxedString[T ~string](v *T) *testkube.BoxedString {
	if v == nil {
		return nil
	}
	return &testkube.BoxedString{Value: string(*v)}
}

func MapBoolToBoxedBoolean(v *bool) *testkube.BoxedBoolean {
	if v == nil {
		return nil
	}
	return &testkube.BoxedBoolean{Value: *v}
}

func MapStringSliceToBoxedStringList(v *[]string) *testkube.BoxedStringList {
	if v == nil {
		return nil
	}
	return &testkube.BoxedStringList{Value: *v}
}

func MapInt64ToBoxedInteger(v *int64) *testkube.BoxedInteger {
	if v == nil {
		return MapInt32ToBoxedInteger(nil)
	}
	return MapInt32ToBoxedInteger(common.Ptr(int32(*v)))
}

func MapInt32ToBoxedInteger(v *int32) *testkube.BoxedInteger {
	if v == nil {
		return nil
	}
	return &testkube.BoxedInteger{Value: *v}
}

func MapQuantityToBoxedString(v *resource.Quantity) *testkube.BoxedString {
	if v == nil {
		return nil
	}
	return &testkube.BoxedString{Value: v.String()}
}

func MapDynamicListMapKubeToAPI(v map[string]testworkflowsv1.DynamicList) map[string]interface{} {
	if len(v) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(v))
	for k := range v {
		result[k] = MapDynamicListKubeToAPI(v[k])
	}
	return result
}

func MapDynamicListKubeToAPI(v testworkflowsv1.DynamicList) interface{} {
	if v.Dynamic {
		return v.Expression
	}
	return v.Static
}

func MapHostPathVolumeSourceKubeToAPI(v corev1.HostPathVolumeSource) testkube.HostPathVolumeSource {
	return testkube.HostPathVolumeSource{
		Path:  v.Path,
		Type_: MapStringTypeToBoxedString[corev1.HostPathType](v.Type),
	}
}

func MapEmptyDirVolumeSourceKubeToAPI(v corev1.EmptyDirVolumeSource) testkube.EmptyDirVolumeSource {
	return testkube.EmptyDirVolumeSource{
		Medium:    string(v.Medium),
		SizeLimit: MapQuantityToBoxedString(v.SizeLimit),
	}
}

func MapGCEPersistentDiskVolumeSourceKubeToAPI(v corev1.GCEPersistentDiskVolumeSource) testkube.GcePersistentDiskVolumeSource {
	return testkube.GcePersistentDiskVolumeSource{
		PdName:    v.PDName,
		FsType:    v.FSType,
		Partition: v.Partition,
		ReadOnly:  v.ReadOnly,
	}
}

func MapAWSElasticBlockStoreVolumeSourceKubeToAPI(v corev1.AWSElasticBlockStoreVolumeSource) testkube.AwsElasticBlockStoreVolumeSource {
	return testkube.AwsElasticBlockStoreVolumeSource{
		VolumeID:  v.VolumeID,
		FsType:    v.FSType,
		Partition: v.Partition,
		ReadOnly:  v.ReadOnly,
	}
}

func MapKeyToPathKubeToAPI(v corev1.KeyToPath) testkube.SecretVolumeSourceItems {
	return testkube.SecretVolumeSourceItems{
		Key:  v.Key,
		Path: v.Path,
		Mode: MapInt32ToBoxedInteger(v.Mode),
	}
}

func MapSecretVolumeSourceKubeToAPI(v corev1.SecretVolumeSource) testkube.SecretVolumeSource {
	return testkube.SecretVolumeSource{
		SecretName:  v.SecretName,
		Items:       common.MapSlice(v.Items, MapKeyToPathKubeToAPI),
		DefaultMode: MapInt32ToBoxedInteger(v.DefaultMode),
		Optional:    common.ResolvePtr(v.Optional, false),
	}
}

func MapNFSVolumeSourceKubeToAPI(v corev1.NFSVolumeSource) testkube.NfsVolumeSource {
	return testkube.NfsVolumeSource{
		Server:   v.Server,
		Path:     v.Path,
		ReadOnly: v.ReadOnly,
	}
}

func MapPersistentVolumeClaimVolumeSourceKubeToAPI(v corev1.PersistentVolumeClaimVolumeSource) testkube.PersistentVolumeClaimVolumeSource {
	return testkube.PersistentVolumeClaimVolumeSource{
		ClaimName: v.ClaimName,
		ReadOnly:  v.ReadOnly,
	}
}

func MapCephFSVolumeSourceKubeToAPI(v corev1.CephFSVolumeSource) testkube.CephFsVolumeSource {
	return testkube.CephFsVolumeSource{
		Monitors:   v.Monitors,
		Path:       v.Path,
		User:       v.User,
		SecretFile: v.SecretFile,
		SecretRef:  common.MapPtr(v.SecretRef, MapLocalObjectReferenceKubeToAPI),
		ReadOnly:   v.ReadOnly,
	}
}

func MapAzureFileVolumeSourceKubeToAPI(v corev1.AzureFileVolumeSource) testkube.AzureFileVolumeSource {
	return testkube.AzureFileVolumeSource{
		SecretName: v.SecretName,
		ShareName:  v.ShareName,
		ReadOnly:   v.ReadOnly,
	}
}

func MapConfigMapVolumeSourceKubeToAPI(v corev1.ConfigMapVolumeSource) testkube.ConfigMapVolumeSource {
	return testkube.ConfigMapVolumeSource{
		Name:        v.Name,
		Items:       common.MapSlice(v.Items, MapKeyToPathKubeToAPI),
		DefaultMode: MapInt32ToBoxedInteger(v.DefaultMode),
		Optional:    common.ResolvePtr(v.Optional, false),
	}
}

func MapAzureDiskVolumeSourceKubeToAPI(v corev1.AzureDiskVolumeSource) testkube.AzureDiskVolumeSource {
	return testkube.AzureDiskVolumeSource{
		DiskName:    v.DiskName,
		DiskURI:     v.DataDiskURI,
		CachingMode: MapStringTypeToBoxedString[corev1.AzureDataDiskCachingMode](v.CachingMode),
		FsType:      MapStringToBoxedString(v.FSType),
		ReadOnly:    common.ResolvePtr(v.ReadOnly, false),
		Kind:        MapStringTypeToBoxedString[corev1.AzureDataDiskKind](v.Kind),
	}
}

func MapVolumeKubeToAPI(v corev1.Volume) testkube.Volume {
	// TODO: Add rest of VolumeSource types in future,
	//       so they will be recognized by JSON API and persisted with Execution.
	return testkube.Volume{
		Name:                  v.Name,
		HostPath:              common.MapPtr(v.HostPath, MapHostPathVolumeSourceKubeToAPI),
		EmptyDir:              common.MapPtr(v.EmptyDir, MapEmptyDirVolumeSourceKubeToAPI),
		GcePersistentDisk:     common.MapPtr(v.GCEPersistentDisk, MapGCEPersistentDiskVolumeSourceKubeToAPI),
		AwsElasticBlockStore:  common.MapPtr(v.AWSElasticBlockStore, MapAWSElasticBlockStoreVolumeSourceKubeToAPI),
		Secret:                common.MapPtr(v.Secret, MapSecretVolumeSourceKubeToAPI),
		Nfs:                   common.MapPtr(v.NFS, MapNFSVolumeSourceKubeToAPI),
		PersistentVolumeClaim: common.MapPtr(v.PersistentVolumeClaim, MapPersistentVolumeClaimVolumeSourceKubeToAPI),
		Cephfs:                common.MapPtr(v.CephFS, MapCephFSVolumeSourceKubeToAPI),
		AzureFile:             common.MapPtr(v.AzureFile, MapAzureFileVolumeSourceKubeToAPI),
		ConfigMap:             common.MapPtr(v.ConfigMap, MapConfigMapVolumeSourceKubeToAPI),
		AzureDisk:             common.MapPtr(v.AzureDisk, MapAzureDiskVolumeSourceKubeToAPI),
	}
}

func MapEnvVarKubeToAPI(v corev1.EnvVar) testkube.EnvVar {
	return testkube.EnvVar{
		Name:      v.Name,
		Value:     v.Value,
		ValueFrom: common.MapPtr(v.ValueFrom, MapEnvVarSourceKubeToAPI),
	}
}

func MapConfigMapKeyRefKubeToAPI(v *corev1.ConfigMapKeySelector) *testkube.EnvVarSourceConfigMapKeyRef {
	if v == nil {
		return nil
	}
	return &testkube.EnvVarSourceConfigMapKeyRef{
		Key:      v.Key,
		Name:     v.Name,
		Optional: common.ResolvePtr(v.Optional, false),
	}
}

func MapFieldRefKubeToAPI(v *corev1.ObjectFieldSelector) *testkube.EnvVarSourceFieldRef {
	if v == nil {
		return nil
	}
	return &testkube.EnvVarSourceFieldRef{
		ApiVersion: v.APIVersion,
		FieldPath:  v.FieldPath,
	}
}

func MapResourceFieldRefKubeToAPI(v *corev1.ResourceFieldSelector) *testkube.EnvVarSourceResourceFieldRef {
	if v == nil {
		return nil
	}
	divisor := ""
	if !v.Divisor.IsZero() {
		divisor = v.Divisor.String()
	}
	return &testkube.EnvVarSourceResourceFieldRef{
		ContainerName: v.ContainerName,
		Divisor:       divisor,
		Resource:      v.Resource,
	}
}

func MapSecretKeyRefKubeToAPI(v *corev1.SecretKeySelector) *testkube.EnvVarSourceSecretKeyRef {
	if v == nil {
		return nil
	}
	return &testkube.EnvVarSourceSecretKeyRef{
		Key:      v.Key,
		Name:     v.Name,
		Optional: common.ResolvePtr(v.Optional, false),
	}
}

func MapEnvVarSourceKubeToAPI(v corev1.EnvVarSource) testkube.EnvVarSource {
	return testkube.EnvVarSource{
		ConfigMapKeyRef:  MapConfigMapKeyRefKubeToAPI(v.ConfigMapKeyRef),
		FieldRef:         MapFieldRefKubeToAPI(v.FieldRef),
		ResourceFieldRef: MapResourceFieldRefKubeToAPI(v.ResourceFieldRef),
		SecretKeyRef:     MapSecretKeyRefKubeToAPI(v.SecretKeyRef),
	}
}

func MapConfigMapEnvSourceKubeToAPI(v *corev1.ConfigMapEnvSource) *testkube.ConfigMapEnvSource {
	if v == nil {
		return nil
	}
	return &testkube.ConfigMapEnvSource{
		Name:     v.Name,
		Optional: common.ResolvePtr(v.Optional, false),
	}
}

func MapSecretEnvSourceKubeToAPI(v *corev1.SecretEnvSource) *testkube.SecretEnvSource {
	if v == nil {
		return nil
	}
	return &testkube.SecretEnvSource{
		Name:     v.Name,
		Optional: common.ResolvePtr(v.Optional, false),
	}
}

func MapEnvFromSourceKubeToAPI(v corev1.EnvFromSource) testkube.EnvFromSource {
	return testkube.EnvFromSource{
		Prefix:       v.Prefix,
		ConfigMapRef: MapConfigMapEnvSourceKubeToAPI(v.ConfigMapRef),
		SecretRef:    MapSecretEnvSourceKubeToAPI(v.SecretRef),
	}
}

func MapSecurityContextKubeToAPI(v *corev1.SecurityContext) *testkube.SecurityContext {
	if v == nil {
		return nil
	}
	return &testkube.SecurityContext{
		Privileged:               MapBoolToBoxedBoolean(v.Privileged),
		RunAsUser:                MapInt64ToBoxedInteger(v.RunAsUser),
		RunAsGroup:               MapInt64ToBoxedInteger(v.RunAsGroup),
		RunAsNonRoot:             MapBoolToBoxedBoolean(v.RunAsNonRoot),
		ReadOnlyRootFilesystem:   MapBoolToBoxedBoolean(v.ReadOnlyRootFilesystem),
		AllowPrivilegeEscalation: MapBoolToBoxedBoolean(v.AllowPrivilegeEscalation),
	}
}

func MapLocalObjectReferenceKubeToAPI(v corev1.LocalObjectReference) testkube.LocalObjectReference {
	return testkube.LocalObjectReference{Name: v.Name}
}

func MapConfigValueKubeToAPI(v map[string]intstr.IntOrString) map[string]string {
	return common.MapMap(v, MapIntOrStringToString)
}

func MapParameterTypeKubeToAPI(v testworkflowsv1.ParameterType) *testkube.TestWorkflowParameterType {
	if v == "" {
		return nil
	}
	return common.Ptr(testkube.TestWorkflowParameterType(v))
}

func MapGitAuthTypeKubeToAPI(v testsv3.GitAuthType) *testkube.ContentGitAuthType {
	if v == "" {
		return nil
	}
	return common.Ptr(testkube.ContentGitAuthType(v))
}

func MapImagePullPolicyKubeToAPI(v corev1.PullPolicy) *testkube.ImagePullPolicy {
	if v == "" {
		return nil
	}
	return common.Ptr(testkube.ImagePullPolicy(v))
}

func MapParameterSchemaKubeToAPI(v testworkflowsv1.ParameterSchema) testkube.TestWorkflowParameterSchema {
	return testkube.TestWorkflowParameterSchema{
		Description:      v.Description,
		Type_:            MapParameterTypeKubeToAPI(v.Type),
		Enum:             v.Enum,
		Example:          common.ResolvePtr(common.MapPtr(v.Example, MapIntOrStringToString), ""),
		Default_:         MapStringToBoxedString(MapIntOrStringPtrToStringPtr(v.Default)),
		Format:           v.Format,
		Pattern:          v.Pattern,
		MinLength:        MapInt64ToBoxedInteger(v.MinLength),
		MaxLength:        MapInt64ToBoxedInteger(v.MaxLength),
		Minimum:          MapInt64ToBoxedInteger(v.Minimum),
		Maximum:          MapInt64ToBoxedInteger(v.Maximum),
		ExclusiveMinimum: MapInt64ToBoxedInteger(v.ExclusiveMinimum),
		ExclusiveMaximum: MapInt64ToBoxedInteger(v.ExclusiveMaximum),
		MultipleOf:       MapInt64ToBoxedInteger(v.MultipleOf),
	}
}

func MapTemplateRefKubeToAPI(v testworkflowsv1.TemplateRef) testkube.TestWorkflowTemplateRef {
	return testkube.TestWorkflowTemplateRef{
		Name:   v.Name,
		Config: MapConfigValueKubeToAPI(v.Config),
	}
}

func MapContentGitKubeToAPI(v testworkflowsv1.ContentGit) testkube.TestWorkflowContentGit {
	return testkube.TestWorkflowContentGit{
		Uri:          v.Uri,
		Revision:     v.Revision,
		Username:     v.Username,
		UsernameFrom: common.MapPtr(v.UsernameFrom, MapEnvVarSourceKubeToAPI),
		Token:        v.Token,
		TokenFrom:    common.MapPtr(v.TokenFrom, MapEnvVarSourceKubeToAPI),
		AuthType:     MapGitAuthTypeKubeToAPI(v.AuthType),
		MountPath:    v.MountPath,
		Paths:        v.Paths,
	}
}

func MapContentTarballKubeToAPI(v testworkflowsv1.ContentTarball) testkube.TestWorkflowContentTarball {
	return testkube.TestWorkflowContentTarball{
		Url:   v.Url,
		Path:  v.Path,
		Mount: MapBoolToBoxedBoolean(v.Mount),
	}
}

func MapContentKubeToAPI(v testworkflowsv1.Content) testkube.TestWorkflowContent {
	return testkube.TestWorkflowContent{
		Git:     common.MapPtr(v.Git, MapContentGitKubeToAPI),
		Files:   common.MapSlice(v.Files, MapContentFileKubeToAPI),
		Tarball: common.MapSlice(v.Tarball, MapContentTarballKubeToAPI),
	}
}

func MapContentFileKubeToAPI(v testworkflowsv1.ContentFile) testkube.TestWorkflowContentFile {
	return testkube.TestWorkflowContentFile{
		Path:        v.Path,
		Content:     v.Content,
		ContentFrom: common.MapPtr(v.ContentFrom, MapEnvVarSourceKubeToAPI),
		Mode:        MapInt32ToBoxedInteger(v.Mode),
	}
}

func MapResourcesListKubeToAPI(v map[corev1.ResourceName]intstr.IntOrString) *testkube.TestWorkflowResourcesList {
	if len(v) == 0 {
		return nil
	}
	empty := intstr.IntOrString{Type: intstr.String, StrVal: ""}
	return &testkube.TestWorkflowResourcesList{
		Cpu:              MapIntOrStringToString(common.GetMapValue(v, corev1.ResourceCPU, empty)),
		Memory:           MapIntOrStringToString(common.GetMapValue(v, corev1.ResourceMemory, empty)),
		Storage:          MapIntOrStringToString(common.GetMapValue(v, corev1.ResourceStorage, empty)),
		EphemeralStorage: MapIntOrStringToString(common.GetMapValue(v, corev1.ResourceEphemeralStorage, empty)),
	}
}

func MapResourcesKubeToAPI(v testworkflowsv1.Resources) testkube.TestWorkflowResources {
	requests := MapResourcesListKubeToAPI(v.Requests)
	limits := MapResourcesListKubeToAPI(v.Limits)
	return testkube.TestWorkflowResources{
		Limits:   limits,
		Requests: requests,
	}
}

func MapJobConfigKubeToAPI(v testworkflowsv1.JobConfig) testkube.TestWorkflowJobConfig {
	return testkube.TestWorkflowJobConfig{
		Labels:                v.Labels,
		Annotations:           v.Annotations,
		Namespace:             v.Namespace,
		ActiveDeadlineSeconds: MapInt64ToBoxedInteger(v.ActiveDeadlineSeconds),
	}
}

func MapEventKubeToAPI(v testworkflowsv1.Event) testkube.TestWorkflowEvent {
	return testkube.TestWorkflowEvent{
		Cronjob: common.MapPtr(v.Cronjob, MapCronJobConfigKubeToAPI),
	}
}

func MapCronJobConfigKubeToAPI(v testworkflowsv1.CronJobConfig) testkube.TestWorkflowCronJobConfig {
	return testkube.TestWorkflowCronJobConfig{
		Cron:        v.Cron,
		Labels:      v.Labels,
		Annotations: v.Annotations,
	}
}

func MapTolerationKubeToAPI(v corev1.Toleration) testkube.Toleration {
	return testkube.Toleration{
		Key:               v.Key,
		Operator:          common.MapEnumToString(v.Operator),
		Value:             v.Value,
		Effect:            common.MapEnumToString(v.Effect),
		TolerationSeconds: MapInt64ToBoxedInteger(v.TolerationSeconds),
	}
}

func MapHostAliasKubeToAPI(v corev1.HostAlias) testkube.HostAlias {
	return testkube.HostAlias{Ip: v.IP, Hostnames: v.Hostnames}
}

func MapTopologySpreadConstraintKubeToAPI(v corev1.TopologySpreadConstraint) testkube.TopologySpreadConstraint {
	return testkube.TopologySpreadConstraint{
		MaxSkew:            v.MaxSkew,
		TopologyKey:        v.TopologyKey,
		WhenUnsatisfiable:  common.MapEnumToString(v.WhenUnsatisfiable),
		LabelSelector:      common.MapPtr(v.LabelSelector, MapLabelSelectorKubeToAPI),
		MinDomains:         MapInt32ToBoxedInteger(v.MinDomains),
		NodeAffinityPolicy: MapStringToBoxedString(common.MapPtr(v.NodeAffinityPolicy, common.MapEnumToString[corev1.NodeInclusionPolicy])),
		NodeTaintsPolicy:   MapStringToBoxedString(common.MapPtr(v.NodeTaintsPolicy, common.MapEnumToString[corev1.NodeInclusionPolicy])),
		MatchLabelKeys:     v.MatchLabelKeys,
	}
}

func MapPodSchedulingGateKubeToAPI(v corev1.PodSchedulingGate) testkube.PodSchedulingGate {
	return testkube.PodSchedulingGate{Name: v.Name}
}

func MapPodResourceClaimKubeToAPI(v corev1.PodResourceClaim) testkube.PodResourceClaim {
	return testkube.PodResourceClaim{
		Name: v.Name,
		Source: &testkube.ClaimSource{
			ResourceClaimName:         MapStringToBoxedString(v.Source.ResourceClaimName),
			ResourceClaimTemplateName: MapStringToBoxedString(v.Source.ResourceClaimTemplateName),
		},
	}
}

func MapPodSecurityContextKubeToAPI(v corev1.PodSecurityContext) testkube.PodSecurityContext {
	return testkube.PodSecurityContext{
		RunAsUser:    MapInt64ToBoxedInteger(v.RunAsUser),
		RunAsGroup:   MapInt64ToBoxedInteger(v.RunAsGroup),
		RunAsNonRoot: MapBoolToBoxedBoolean(v.RunAsNonRoot),
	}
}

func MapNodeSelectorRequirementKubeToAPI(v corev1.NodeSelectorRequirement) testkube.NodeSelectorRequirement {
	return testkube.NodeSelectorRequirement{
		Key:      v.Key,
		Operator: common.MapEnumToString(v.Operator),
		Values:   v.Values,
	}
}

func MapNodeSelectorTermKubeToAPI(v corev1.NodeSelectorTerm) testkube.NodeSelectorTerm {
	return testkube.NodeSelectorTerm{
		MatchExpressions: common.MapSlice(v.MatchExpressions, MapNodeSelectorRequirementKubeToAPI),
		MatchFields:      common.MapSlice(v.MatchFields, MapNodeSelectorRequirementKubeToAPI),
	}
}

func MapNodeSelectorKubeToAPI(v corev1.NodeSelector) testkube.NodeSelector {
	return testkube.NodeSelector{
		NodeSelectorTerms: common.MapSlice(v.NodeSelectorTerms, MapNodeSelectorTermKubeToAPI),
	}
}

func MapPreferredSchedulingTermKubeToAPI(v corev1.PreferredSchedulingTerm) testkube.PreferredSchedulingTerm {
	return testkube.PreferredSchedulingTerm{
		Weight:     v.Weight,
		Preference: common.Ptr(MapNodeSelectorTermKubeToAPI(v.Preference)),
	}
}

func MapNodeAffinityKubeToAPI(v corev1.NodeAffinity) testkube.NodeAffinity {
	return testkube.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution:  common.MapPtr(v.RequiredDuringSchedulingIgnoredDuringExecution, MapNodeSelectorKubeToAPI),
		PreferredDuringSchedulingIgnoredDuringExecution: common.MapSlice(v.PreferredDuringSchedulingIgnoredDuringExecution, MapPreferredSchedulingTermKubeToAPI),
	}
}

func MapPodAffinityKubeToAPI(v corev1.PodAffinity) testkube.PodAffinity {
	return testkube.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution:  common.MapSlice(v.RequiredDuringSchedulingIgnoredDuringExecution, MapPodAffinityTermKubeToAPI),
		PreferredDuringSchedulingIgnoredDuringExecution: common.MapSlice(v.PreferredDuringSchedulingIgnoredDuringExecution, MapWeightedPodAffinityTermKubeToAPI),
	}
}

func MapLabelSelectorRequirementKubeToAPI(v metav1.LabelSelectorRequirement) testkube.LabelSelectorRequirement {
	return testkube.LabelSelectorRequirement{
		Key:      v.Key,
		Operator: common.MapEnumToString(v.Operator),
		Values:   v.Values,
	}
}

func MapLabelSelectorKubeToAPI(v metav1.LabelSelector) testkube.LabelSelector {
	return testkube.LabelSelector{
		MatchLabels:      v.MatchLabels,
		MatchExpressions: common.MapSlice(v.MatchExpressions, MapLabelSelectorRequirementKubeToAPI),
	}
}

func MapPodAffinityTermKubeToAPI(v corev1.PodAffinityTerm) testkube.PodAffinityTerm {
	return testkube.PodAffinityTerm{
		LabelSelector:     common.MapPtr(v.LabelSelector, MapLabelSelectorKubeToAPI),
		Namespaces:        v.Namespaces,
		TopologyKey:       v.TopologyKey,
		NamespaceSelector: common.MapPtr(v.NamespaceSelector, MapLabelSelectorKubeToAPI),
	}
}

func MapWeightedPodAffinityTermKubeToAPI(v corev1.WeightedPodAffinityTerm) testkube.WeightedPodAffinityTerm {
	return testkube.WeightedPodAffinityTerm{
		Weight:          v.Weight,
		PodAffinityTerm: common.Ptr(MapPodAffinityTermKubeToAPI(v.PodAffinityTerm)),
	}
}

func MapPodAntiAffinityKubeToAPI(v corev1.PodAntiAffinity) testkube.PodAffinity {
	return testkube.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution:  common.MapSlice(v.RequiredDuringSchedulingIgnoredDuringExecution, MapPodAffinityTermKubeToAPI),
		PreferredDuringSchedulingIgnoredDuringExecution: common.MapSlice(v.PreferredDuringSchedulingIgnoredDuringExecution, MapWeightedPodAffinityTermKubeToAPI),
	}
}

func MapAffinityKubeToAPI(v corev1.Affinity) testkube.Affinity {
	return testkube.Affinity{
		NodeAffinity:    common.MapPtr(v.NodeAffinity, MapNodeAffinityKubeToAPI),
		PodAffinity:     common.MapPtr(v.PodAffinity, MapPodAffinityKubeToAPI),
		PodAntiAffinity: common.MapPtr(v.PodAntiAffinity, MapPodAntiAffinityKubeToAPI),
	}
}

func MapPodDNSConfigOptionKubeToAPI(v corev1.PodDNSConfigOption) testkube.PodDnsConfigOption {
	return testkube.PodDnsConfigOption{
		Name:  v.Name,
		Value: MapStringToBoxedString(v.Value),
	}
}

func MapPodDNSConfigKubeToAPI(v corev1.PodDNSConfig) testkube.PodDnsConfig {
	return testkube.PodDnsConfig{
		Nameservers: v.Nameservers,
		Searches:    v.Searches,
		Options:     common.MapSlice(v.Options, MapPodDNSConfigOptionKubeToAPI),
	}
}

func MapPodConfigKubeToAPI(v testworkflowsv1.PodConfig) testkube.TestWorkflowPodConfig {
	return testkube.TestWorkflowPodConfig{
		ServiceAccountName:        v.ServiceAccountName,
		ImagePullSecrets:          common.MapSlice(v.ImagePullSecrets, MapLocalObjectReferenceKubeToAPI),
		NodeSelector:              v.NodeSelector,
		Labels:                    v.Labels,
		Annotations:               v.Annotations,
		Volumes:                   common.MapSlice(v.Volumes, MapVolumeKubeToAPI),
		ActiveDeadlineSeconds:     MapInt64ToBoxedInteger(v.ActiveDeadlineSeconds),
		DnsPolicy:                 common.MapEnumToString(v.DNSPolicy),
		NodeName:                  v.NodeName,
		SecurityContext:           common.MapPtr(v.SecurityContext, MapPodSecurityContextKubeToAPI),
		Hostname:                  v.Hostname,
		Subdomain:                 v.Subdomain,
		Affinity:                  common.MapPtr(v.Affinity, MapAffinityKubeToAPI),
		Tolerations:               common.MapSlice(v.Tolerations, MapTolerationKubeToAPI),
		HostAliases:               common.MapSlice(v.HostAliases, MapHostAliasKubeToAPI),
		PriorityClassName:         v.PriorityClassName,
		Priority:                  MapInt32ToBoxedInteger(v.Priority),
		DnsConfig:                 common.MapPtr(v.DNSConfig, MapPodDNSConfigKubeToAPI),
		PreemptionPolicy:          MapStringToBoxedString(common.MapPtr(v.PreemptionPolicy, common.MapEnumToString[corev1.PreemptionPolicy])),
		TopologySpreadConstraints: common.MapSlice(v.TopologySpreadConstraints, MapTopologySpreadConstraintKubeToAPI),
		SchedulingGates:           common.MapSlice(v.SchedulingGates, MapPodSchedulingGateKubeToAPI),
		ResourceClaims:            common.MapSlice(v.ResourceClaims, MapPodResourceClaimKubeToAPI),
	}
}

func MapVolumeMountKubeToAPI(v corev1.VolumeMount) testkube.VolumeMount {
	return testkube.VolumeMount{
		Name:             v.Name,
		ReadOnly:         v.ReadOnly,
		MountPath:        v.MountPath,
		SubPath:          v.SubPath,
		MountPropagation: MapStringTypeToBoxedString[corev1.MountPropagationMode](v.MountPropagation),
		SubPathExpr:      v.SubPathExpr,
	}
}

func MapContainerConfigKubeToAPI(v testworkflowsv1.ContainerConfig) testkube.TestWorkflowContainerConfig {
	return testkube.TestWorkflowContainerConfig{
		WorkingDir:      MapStringToBoxedString(v.WorkingDir),
		Image:           v.Image,
		ImagePullPolicy: MapImagePullPolicyKubeToAPI(v.ImagePullPolicy),
		Env:             common.MapSlice(v.Env, MapEnvVarKubeToAPI),
		EnvFrom:         common.MapSlice(v.EnvFrom, MapEnvFromSourceKubeToAPI),
		Command:         MapStringSliceToBoxedStringList(v.Command),
		Args:            MapStringSliceToBoxedStringList(v.Args),
		Resources:       common.MapPtr(v.Resources, MapResourcesKubeToAPI),
		SecurityContext: MapSecurityContextKubeToAPI(v.SecurityContext),
		VolumeMounts:    common.MapSlice(v.VolumeMounts, MapVolumeMountKubeToAPI),
	}
}

func MapStepRunKubeToAPI(v testworkflowsv1.StepRun) testkube.TestWorkflowStepRun {
	return testkube.TestWorkflowStepRun{
		WorkingDir:      MapStringToBoxedString(v.WorkingDir),
		Image:           v.Image,
		ImagePullPolicy: MapImagePullPolicyKubeToAPI(v.ImagePullPolicy),
		Env:             common.MapSlice(v.Env, MapEnvVarKubeToAPI),
		EnvFrom:         common.MapSlice(v.EnvFrom, MapEnvFromSourceKubeToAPI),
		Command:         MapStringSliceToBoxedStringList(v.Command),
		Args:            MapStringSliceToBoxedStringList(v.Args),
		Shell:           MapStringToBoxedString(v.Shell),
		Resources:       common.MapPtr(v.Resources, MapResourcesKubeToAPI),
		SecurityContext: MapSecurityContextKubeToAPI(v.SecurityContext),
		VolumeMounts:    common.MapSlice(v.VolumeMounts, MapVolumeMountKubeToAPI),
	}
}

func MapTestVariableKubeToAPI(v testsv3.Variable) testkube.Variable {
	var configMapRef *testkube.ConfigMapRef
	if v.ValueFrom.ConfigMapKeyRef != nil {
		configMapRef = &testkube.ConfigMapRef{
			Name: v.ValueFrom.ConfigMapKeyRef.Name,
			Key:  v.ValueFrom.ConfigMapKeyRef.Key,
		}
	}
	var secretRef *testkube.SecretRef
	if v.ValueFrom.SecretKeyRef != nil {
		secretRef = &testkube.SecretRef{
			Name: v.ValueFrom.SecretKeyRef.Name,
			Key:  v.ValueFrom.SecretKeyRef.Key,
		}
	}
	return testkube.Variable{
		Type_:        common.PtrOrNil(testkube.VariableType(v.Type_)),
		Name:         v.Name,
		Value:        v.Value,
		SecretRef:    secretRef,
		ConfigMapRef: configMapRef,
	}
}

func MapTestArtifactRequestKubeToAPI(v testsv3.ArtifactRequest) testkube.ArtifactRequest {
	return testkube.ArtifactRequest{
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

func MapTestEnvReferenceKubeToAPI(v testsv3.EnvReference) testkube.EnvReference {
	return testkube.EnvReference{
		Reference:      common.PtrOrNil(testkube.LocalObjectReference{Name: v.Name}),
		Mount:          v.Mount,
		MountPath:      v.MountPath,
		MapToVariables: v.MapToVariables,
	}
}

func MapStepExecuteTestExecutionRequestKubeToAPI(v testworkflowsv1.TestExecutionRequest) testkube.TestWorkflowStepExecuteTestExecutionRequest {
	return testkube.TestWorkflowStepExecuteTestExecutionRequest{
		Name:                               v.Name,
		ExecutionLabels:                    v.ExecutionLabels,
		VariablesFile:                      v.VariablesFile,
		IsVariablesFileUploaded:            v.IsVariablesFileUploaded,
		Variables:                          common.MapMap(v.Variables, MapTestVariableKubeToAPI),
		TestSecretUUID:                     v.TestSecretUUID,
		Args:                               v.Args,
		ArgsMode:                           string(v.ArgsMode),
		Command:                            v.Command,
		Image:                              v.Image,
		ImagePullSecrets:                   common.MapSlice(v.ImagePullSecrets, MapLocalObjectReferenceKubeToAPI),
		Sync:                               v.Sync,
		HttpProxy:                          v.HttpProxy,
		HttpsProxy:                         v.HttpsProxy,
		NegativeTest:                       v.NegativeTest,
		ActiveDeadlineSeconds:              v.ActiveDeadlineSeconds,
		ArtifactRequest:                    common.MapPtr(v.ArtifactRequest, MapTestArtifactRequestKubeToAPI),
		JobTemplate:                        v.JobTemplate,
		CronJobTemplate:                    v.CronJobTemplate,
		PreRunScript:                       v.PreRunScript,
		PostRunScript:                      v.PostRunScript,
		ExecutePostRunScriptBeforeScraping: v.ExecutePostRunScriptBeforeScraping,
		SourceScripts:                      v.SourceScripts,
		ScraperTemplate:                    v.ScraperTemplate,
		EnvConfigMaps:                      common.MapSlice(v.EnvConfigMaps, MapTestEnvReferenceKubeToAPI),
		EnvSecrets:                         common.MapSlice(v.EnvSecrets, MapTestEnvReferenceKubeToAPI),
		ExecutionNamespace:                 v.ExecutionNamespace,
	}
}

func MapTarballFilePatternKubeToAPI(v testworkflowsv1.DynamicList) testkube.TestWorkflowTarballFilePattern {
	if v.Expression != "" {
		return testkube.TestWorkflowTarballFilePattern{Expression: v.Expression}
	}
	return testkube.TestWorkflowTarballFilePattern{Static: common.MapSlice(v.Static, func(s string) interface{} {
		return s
	})}
}

func MapTarballRequestKubeToAPI(v testworkflowsv1.TarballRequest) testkube.TestWorkflowTarballRequest {
	return testkube.TestWorkflowTarballRequest{
		From:  v.From,
		Files: common.MapPtr(v.Files, MapTarballFilePatternKubeToAPI),
	}
}

func MapStepExecuteTestKubeToAPI(v testworkflowsv1.StepExecuteTest) testkube.TestWorkflowStepExecuteTestRef {
	return testkube.TestWorkflowStepExecuteTestRef{
		Name:             v.Name,
		Description:      v.Description,
		ExecutionRequest: common.MapPtr(v.ExecutionRequest, MapStepExecuteTestExecutionRequestKubeToAPI),
		Tarball:          common.MapMap(v.Tarball, MapTarballRequestKubeToAPI),
		Count:            MapIntOrStringToBoxedString(v.Count),
		MaxCount:         MapIntOrStringToBoxedString(v.MaxCount),
		Matrix:           MapDynamicListMapKubeToAPI(v.Matrix),
		Shards:           MapDynamicListMapKubeToAPI(v.Shards),
	}
}

func MapStepExecuteTestWorkflowKubeToAPI(v testworkflowsv1.StepExecuteWorkflow) testkube.TestWorkflowStepExecuteTestWorkflowRef {
	return testkube.TestWorkflowStepExecuteTestWorkflowRef{
		Name:          v.Name,
		Description:   v.Description,
		ExecutionName: v.ExecutionName,
		Tarball:       common.MapMap(v.Tarball, MapTarballRequestKubeToAPI),
		Config:        MapConfigValueKubeToAPI(v.Config),
		Count:         MapIntOrStringToBoxedString(v.Count),
		MaxCount:      MapIntOrStringToBoxedString(v.MaxCount),
		Matrix:        MapDynamicListMapKubeToAPI(v.Matrix),
		Shards:        MapDynamicListMapKubeToAPI(v.Shards),
	}
}

func MapStepExecuteKubeToAPI(v testworkflowsv1.StepExecute) testkube.TestWorkflowStepExecute {
	return testkube.TestWorkflowStepExecute{
		Parallelism: v.Parallelism,
		Async:       v.Async,
		Tests:       common.MapSlice(v.Tests, MapStepExecuteTestKubeToAPI),
		Workflows:   common.MapSlice(v.Workflows, MapStepExecuteTestWorkflowKubeToAPI),
	}
}

func MapStepArtifactsCompressionKubeToAPI(v testworkflowsv1.ArtifactCompression) testkube.TestWorkflowStepArtifactsCompression {
	return testkube.TestWorkflowStepArtifactsCompression{
		Name: v.Name,
	}
}

func MapStepArtifactsKubeToAPI(v testworkflowsv1.StepArtifacts) testkube.TestWorkflowStepArtifacts {
	return testkube.TestWorkflowStepArtifacts{
		WorkingDir: MapStringToBoxedString(v.WorkingDir),
		Compress:   common.MapPtr(v.Compress, MapStepArtifactsCompressionKubeToAPI),
		Paths:      v.Paths,
	}
}

func MapRetryPolicyKubeToAPI(v testworkflowsv1.RetryPolicy) testkube.TestWorkflowRetryPolicy {
	return testkube.TestWorkflowRetryPolicy{
		Count: v.Count,
		Until: v.Until,
	}
}

func MapStepParallelTransferKubeToAPI(v testworkflowsv1.StepParallelTransfer) testkube.TestWorkflowStepParallelTransfer {
	return testkube.TestWorkflowStepParallelTransfer{
		From:  v.From,
		To:    v.To,
		Files: common.MapPtr(v.Files, MapTarballFilePatternKubeToAPI),
		Mount: MapBoolToBoxedBoolean(v.Mount),
	}
}

func MapStepParallelKubeToAPI(v testworkflowsv1.StepParallel) testkube.TestWorkflowStepParallel {
	return testkube.TestWorkflowStepParallel{
		Count:     MapIntOrStringToBoxedString(v.Count),
		MaxCount:  MapIntOrStringToBoxedString(v.MaxCount),
		Matrix:    MapDynamicListMapKubeToAPI(v.Matrix),
		Shards:    MapDynamicListMapKubeToAPI(v.Shards),
		Transfer:  common.MapSlice(v.Transfer, MapStepParallelTransferKubeToAPI),
		Use:       common.MapSlice(v.Use, MapTemplateRefKubeToAPI),
		Config:    common.MapMap(v.Config, MapParameterSchemaKubeToAPI),
		Content:   common.MapPtr(v.Content, MapContentKubeToAPI),
		Container: common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Job:       common.MapPtr(v.Job, MapJobConfigKubeToAPI),
		Pod:       common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
		Setup:     common.MapSlice(v.Setup, MapStepKubeToAPI),
		Steps:     common.MapSlice(v.Steps, MapStepKubeToAPI),
		After:     common.MapSlice(v.After, MapStepKubeToAPI),
		Events:    common.MapSlice(v.Events, MapEventKubeToAPI),
	}
}

func MapIndependentStepParallelKubeToAPI(v testworkflowsv1.IndependentStepParallel) testkube.TestWorkflowIndependentStepParallel {
	return testkube.TestWorkflowIndependentStepParallel{
		Count:     MapIntOrStringToBoxedString(v.Count),
		MaxCount:  MapIntOrStringToBoxedString(v.MaxCount),
		Matrix:    MapDynamicListMapKubeToAPI(v.Matrix),
		Shards:    MapDynamicListMapKubeToAPI(v.Shards),
		Transfer:  common.MapSlice(v.Transfer, MapStepParallelTransferKubeToAPI),
		Config:    common.MapMap(v.Config, MapParameterSchemaKubeToAPI),
		Content:   common.MapPtr(v.Content, MapContentKubeToAPI),
		Container: common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Job:       common.MapPtr(v.Job, MapJobConfigKubeToAPI),
		Pod:       common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
		Setup:     common.MapSlice(v.Setup, MapIndependentStepKubeToAPI),
		Steps:     common.MapSlice(v.Steps, MapIndependentStepKubeToAPI),
		After:     common.MapSlice(v.After, MapIndependentStepKubeToAPI),
		Events:    common.MapSlice(v.Events, MapEventKubeToAPI),
	}
}

func MapStepKubeToAPI(v testworkflowsv1.Step) testkube.TestWorkflowStep {
	return testkube.TestWorkflowStep{
		Name:       v.Name,
		Condition:  v.Condition,
		Paused:     v.Paused,
		Negative:   v.Negative,
		Optional:   v.Optional,
		Use:        common.MapSlice(v.Use, MapTemplateRefKubeToAPI),
		Template:   common.MapPtr(v.Template, MapTemplateRefKubeToAPI),
		Retry:      common.MapPtr(v.Retry, MapRetryPolicyKubeToAPI),
		Timeout:    v.Timeout,
		Delay:      v.Delay,
		Content:    common.MapPtr(v.Content, MapContentKubeToAPI),
		Shell:      v.Shell,
		Run:        common.MapPtr(v.Run, MapStepRunKubeToAPI),
		WorkingDir: MapStringToBoxedString(v.WorkingDir),
		Container:  common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Execute:    common.MapPtr(v.Execute, MapStepExecuteKubeToAPI),
		Artifacts:  common.MapPtr(v.Artifacts, MapStepArtifactsKubeToAPI),
		Setup:      common.MapSlice(v.Setup, MapStepKubeToAPI),
		Steps:      common.MapSlice(v.Steps, MapStepKubeToAPI),
		Parallel:   common.MapPtr(v.Parallel, MapStepParallelKubeToAPI),
	}
}

func MapIndependentStepKubeToAPI(v testworkflowsv1.IndependentStep) testkube.TestWorkflowIndependentStep {
	return testkube.TestWorkflowIndependentStep{
		Name:       v.Name,
		Condition:  v.Condition,
		Paused:     v.Paused,
		Negative:   v.Negative,
		Optional:   v.Optional,
		Retry:      common.MapPtr(v.Retry, MapRetryPolicyKubeToAPI),
		Timeout:    v.Timeout,
		Delay:      v.Delay,
		Content:    common.MapPtr(v.Content, MapContentKubeToAPI),
		Shell:      v.Shell,
		Run:        common.MapPtr(v.Run, MapStepRunKubeToAPI),
		WorkingDir: MapStringToBoxedString(v.WorkingDir),
		Container:  common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Execute:    common.MapPtr(v.Execute, MapStepExecuteKubeToAPI),
		Artifacts:  common.MapPtr(v.Artifacts, MapStepArtifactsKubeToAPI),
		Setup:      common.MapSlice(v.Setup, MapIndependentStepKubeToAPI),
		Steps:      common.MapSlice(v.Steps, MapIndependentStepKubeToAPI),
		Parallel:   common.MapPtr(v.Parallel, MapIndependentStepParallelKubeToAPI),
	}
}

func MapSpecKubeToAPI(v testworkflowsv1.TestWorkflowSpec) testkube.TestWorkflowSpec {
	return testkube.TestWorkflowSpec{
		Use:       common.MapSlice(v.Use, MapTemplateRefKubeToAPI),
		Config:    common.MapMap(v.Config, MapParameterSchemaKubeToAPI),
		Content:   common.MapPtr(v.Content, MapContentKubeToAPI),
		Container: common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Job:       common.MapPtr(v.Job, MapJobConfigKubeToAPI),
		Pod:       common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
		Setup:     common.MapSlice(v.Setup, MapStepKubeToAPI),
		Steps:     common.MapSlice(v.Steps, MapStepKubeToAPI),
		After:     common.MapSlice(v.After, MapStepKubeToAPI),
		Events:    common.MapSlice(v.Events, MapEventKubeToAPI),
	}
}

func MapTemplateSpecKubeToAPI(v testworkflowsv1.TestWorkflowTemplateSpec) testkube.TestWorkflowTemplateSpec {
	return testkube.TestWorkflowTemplateSpec{
		Config:    common.MapMap(v.Config, MapParameterSchemaKubeToAPI),
		Content:   common.MapPtr(v.Content, MapContentKubeToAPI),
		Container: common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Job:       common.MapPtr(v.Job, MapJobConfigKubeToAPI),
		Pod:       common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
		Setup:     common.MapSlice(v.Setup, MapIndependentStepKubeToAPI),
		Steps:     common.MapSlice(v.Steps, MapIndependentStepKubeToAPI),
		After:     common.MapSlice(v.After, MapIndependentStepKubeToAPI),
		Events:    common.MapSlice(v.Events, MapEventKubeToAPI),
	}
}

func MapTestWorkflowKubeToAPI(w testworkflowsv1.TestWorkflow) testkube.TestWorkflow {
	return testkube.TestWorkflow{
		Name:        w.Name,
		Namespace:   w.Namespace,
		Labels:      w.Labels,
		Annotations: w.Annotations,
		Created:     w.CreationTimestamp.Time,
		Description: w.Description,
		Spec:        common.Ptr(MapSpecKubeToAPI(w.Spec)),
	}
}

func MapTestWorkflowTemplateKubeToAPI(w testworkflowsv1.TestWorkflowTemplate) testkube.TestWorkflowTemplate {
	return testkube.TestWorkflowTemplate{
		Name:        w.Name,
		Namespace:   w.Namespace,
		Labels:      w.Labels,
		Annotations: w.Annotations,
		Created:     w.CreationTimestamp.Time,
		Description: w.Description,
		Spec:        common.Ptr(MapTemplateSpecKubeToAPI(w.Spec)),
	}
}

func MapTemplateKubeToAPI(w *testworkflowsv1.TestWorkflowTemplate) *testkube.TestWorkflowTemplate {
	return common.MapPtr(w, MapTestWorkflowTemplateKubeToAPI)
}

func MapKubeToAPI(w *testworkflowsv1.TestWorkflow) *testkube.TestWorkflow {
	return common.MapPtr(w, MapTestWorkflowKubeToAPI)
}

func MapListKubeToAPI(v *testworkflowsv1.TestWorkflowList) []testkube.TestWorkflow {
	workflows := make([]testkube.TestWorkflow, len(v.Items))
	for i, item := range v.Items {
		workflows[i] = MapTestWorkflowKubeToAPI(item)
	}
	return workflows
}

func MapTemplateListKubeToAPI(v *testworkflowsv1.TestWorkflowTemplateList) []testkube.TestWorkflowTemplate {
	workflows := make([]testkube.TestWorkflowTemplate, len(v.Items))
	for i, item := range v.Items {
		workflows[i] = MapTestWorkflowTemplateKubeToAPI(item)
	}
	return workflows
}
