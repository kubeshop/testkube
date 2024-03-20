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

func MapContentKubeToAPI(v testworkflowsv1.Content) testkube.TestWorkflowContent {
	return testkube.TestWorkflowContent{
		Git:   common.MapPtr(v.Git, MapContentGitKubeToAPI),
		Files: common.MapSlice(v.Files, MapContentFileKubeToAPI),
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
		Labels:      v.Labels,
		Annotations: v.Annotations,
	}
}

func MapPodConfigKubeToAPI(v testworkflowsv1.PodConfig) testkube.TestWorkflowPodConfig {
	return testkube.TestWorkflowPodConfig{
		ServiceAccountName: v.ServiceAccountName,
		ImagePullSecrets:   common.MapSlice(v.ImagePullSecrets, MapLocalObjectReferenceKubeToAPI),
		NodeSelector:       v.NodeSelector,
		Labels:             v.Labels,
		Annotations:        v.Annotations,
		Volumes:            common.MapSlice(v.Volumes, MapVolumeKubeToAPI),
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

func MapStepRunKubeToAPI(v testworkflowsv1.StepRun) testkube.TestWorkflowContainerConfig {
	return MapContainerConfigKubeToAPI(v.ContainerConfig)
}

func MapStepExecuteTestKubeToAPI(v testworkflowsv1.StepExecuteTest) testkube.TestWorkflowStepExecuteTestRef {
	return testkube.TestWorkflowStepExecuteTestRef{
		Name: v.Name,
	}
}

func MapTestWorkflowRefKubeToAPI(v testworkflowsv1.StepExecuteWorkflow) testkube.TestWorkflowRef {
	return testkube.TestWorkflowRef{
		Name:   v.Name,
		Config: MapConfigValueKubeToAPI(v.Config),
	}
}

func MapStepExecuteKubeToAPI(v testworkflowsv1.StepExecute) testkube.TestWorkflowStepExecute {
	return testkube.TestWorkflowStepExecute{
		Parallelism: v.Parallelism,
		Async:       v.Async,
		Tests:       common.MapSlice(v.Tests, MapStepExecuteTestKubeToAPI),
		Workflows:   common.MapSlice(v.Workflows, MapTestWorkflowRefKubeToAPI),
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

func MapStepKubeToAPI(v testworkflowsv1.Step) testkube.TestWorkflowStep {
	return testkube.TestWorkflowStep{
		Name:       v.Name,
		Condition:  v.Condition,
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
	}
}

func MapIndependentStepKubeToAPI(v testworkflowsv1.IndependentStep) testkube.TestWorkflowIndependentStep {
	return testkube.TestWorkflowIndependentStep{
		Name:       v.Name,
		Condition:  v.Condition,
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
