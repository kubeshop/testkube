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

func MapCSIVolumeSourceKubeToAPI(v corev1.CSIVolumeSource) testkube.CsiVolumeSource {
	return testkube.CsiVolumeSource{
		Driver:               v.Driver,
		ReadOnly:             MapBoolToBoxedBoolean(v.ReadOnly),
		FsType:               MapStringToBoxedString(v.FSType),
		VolumeAttributes:     v.VolumeAttributes,
		NodePublishSecretRef: common.MapPtr(v.NodePublishSecretRef, MapLocalObjectReferenceKubeToAPI),
	}
}

func MapProjectedVolumeSourceKubeToAPI(v corev1.ProjectedVolumeSource) testkube.ProjectedVolumeSource {
	return testkube.ProjectedVolumeSource{
		DefaultMode: MapInt32ToBoxedInteger(v.DefaultMode),
		Sources:     common.MapSlice(v.Sources, MapVolumeProjectionKubeToAPI),
	}
}

func MapVolumeProjectionKubeToAPI(v corev1.VolumeProjection) testkube.ProjectedVolumeSourceSources {
	return testkube.ProjectedVolumeSourceSources{
		ClusterTrustBundle:  common.MapPtr(v.ClusterTrustBundle, MapClusterTrustBundleProjectionKubeToAPI),
		ConfigMap:           common.MapPtr(v.ConfigMap, MapConfigMapProjectionKubeToAPI),
		DownwardAPI:         common.MapPtr(v.DownwardAPI, MapDownwardAPIProjectionKubeToAPI),
		Secret:              common.MapPtr(v.Secret, MapSecretProjectionKubeToAPI),
		ServiceAccountToken: common.MapPtr(v.ServiceAccountToken, MapServiceAccountTokenProjectionKubeToAPI),
	}
}

func MapConfigMapProjectionKubeToAPI(v corev1.ConfigMapProjection) testkube.ProjectedVolumeSourceConfigMap {
	return testkube.ProjectedVolumeSourceConfigMap{
		Items:    common.MapSlice(v.Items, MapKeyToPathKubeToAPI),
		Name:     v.Name,
		Optional: MapBoolToBoxedBoolean(v.Optional),
	}
}

func MapClusterTrustBundleProjectionKubeToAPI(v corev1.ClusterTrustBundleProjection) testkube.ProjectedVolumeSourceClusterTrustBundle {
	return testkube.ProjectedVolumeSourceClusterTrustBundle{
		LabelSelector: common.MapPtr(v.LabelSelector, MapLabelSelectorKubeToAPI),
		Name:          MapStringToBoxedString(v.Name),
		Optional:      MapBoolToBoxedBoolean(v.Optional),
		Path:          v.Path,
		SignerName:    MapStringToBoxedString(v.SignerName),
	}
}

func MapDownwardAPIProjectionKubeToAPI(v corev1.DownwardAPIProjection) testkube.ProjectedVolumeSourceDownwardApi {
	return testkube.ProjectedVolumeSourceDownwardApi{
		Items: common.MapSlice(v.Items, MapDownwardAPIVolumeFileKubeToAPI),
	}
}

func MapDownwardAPIVolumeFileKubeToAPI(v corev1.DownwardAPIVolumeFile) testkube.ProjectedVolumeSourceDownwardApiItems {
	return testkube.ProjectedVolumeSourceDownwardApiItems{
		FieldRef:         MapFieldRefKubeToAPI(v.FieldRef),
		Mode:             MapInt32ToBoxedInteger(v.Mode),
		Path:             v.Path,
		ResourceFieldRef: MapResourceFieldRefKubeToAPI(v.ResourceFieldRef),
	}
}

func MapSecretProjectionKubeToAPI(v corev1.SecretProjection) testkube.ProjectedVolumeSourceSecret {
	return testkube.ProjectedVolumeSourceSecret{
		Items:    common.MapSlice(v.Items, MapKeyToPathKubeToAPI),
		Name:     v.Name,
		Optional: MapBoolToBoxedBoolean(v.Optional),
	}
}

func MapServiceAccountTokenProjectionKubeToAPI(v corev1.ServiceAccountTokenProjection) testkube.ProjectedVolumeSourceServiceAccountToken {
	return testkube.ProjectedVolumeSourceServiceAccountToken{
		Audience:          v.Audience,
		ExpirationSeconds: MapInt64ToBoxedInteger(v.ExpirationSeconds),
		Path:              v.Path,
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
		Csi:                   common.MapPtr(v.CSI, MapCSIVolumeSourceKubeToAPI),
		Projected:             common.MapPtr(v.Projected, MapProjectedVolumeSourceKubeToAPI),
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

func MapFieldRefKubeToAPI(v *corev1.ObjectFieldSelector) *testkube.FieldRef {
	if v == nil {
		return nil
	}
	return &testkube.FieldRef{
		ApiVersion: v.APIVersion,
		FieldPath:  v.FieldPath,
	}
}

func MapResourceFieldRefKubeToAPI(v *corev1.ResourceFieldSelector) *testkube.ResourceFieldRef {
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
		Sensitive:        v.Sensitive,
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
		SshKey:       v.SshKey,
		SshKeyFrom:   common.MapPtr(v.SshKeyFrom, MapEnvVarSourceKubeToAPI),
		AuthType:     MapGitAuthTypeKubeToAPI(v.AuthType),
		MountPath:    v.MountPath,
		Cone:         v.Cone,
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

func MapTargetKubeToAPI(v testworkflowsv1.Target) testkube.TestWorkflowTarget {
	return testkube.TestWorkflowTarget{
		Match:     v.Match,
		Not:       v.Not,
		Replicate: v.Replicate,
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
		Config:      MapConfigValueKubeToAPI(v.Config),
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
			ResourceClaimName:         MapStringToBoxedString(v.ResourceClaimName),
			ResourceClaimTemplateName: MapStringToBoxedString(v.ResourceClaimTemplateName),
		},
	}
}

func MapPodSecurityContextKubeToAPI(v corev1.PodSecurityContext) testkube.PodSecurityContext {
	return testkube.PodSecurityContext{
		SeLinuxOptions:           common.MapPtr(v.SELinuxOptions, MapSELinuxOptionsKubeToAPI),
		WindowsOptions:           common.MapPtr(v.WindowsOptions, MapWindowsSecurityContextOptionsKubeToAPI),
		RunAsUser:                MapInt64ToBoxedInteger(v.RunAsUser),
		RunAsGroup:               MapInt64ToBoxedInteger(v.RunAsGroup),
		RunAsNonRoot:             MapBoolToBoxedBoolean(v.RunAsNonRoot),
		SupplementalGroups:       v.SupplementalGroups,
		SupplementalGroupsPolicy: MapStringToBoxedString((*string)(v.SupplementalGroupsPolicy)),
		FsGroup:                  MapInt64ToBoxedInteger(v.FSGroup),
		Sysctls:                  common.MapSlice(v.Sysctls, MapSysctlKubeToAPI),
		FsGroupChangePolicy:      MapStringToBoxedString((*string)(v.FSGroupChangePolicy)),
		SeccompProfile:           common.MapPtr(v.SeccompProfile, MapSeccompProfileKubeToAPI),
		AppArmorProfile:          common.MapPtr(v.AppArmorProfile, MapAppArmorProfileKubeToAPI),
		SeLinuxChangePolicy:      MapStringToBoxedString((*string)(v.SELinuxChangePolicy)),
	}
}

func MapSELinuxOptionsKubeToAPI(v corev1.SELinuxOptions) testkube.SeLinuxOptions {
	return testkube.SeLinuxOptions{
		User:  v.User,
		Role:  v.Role,
		Type_: v.Type,
		Level: v.Level,
	}
}

func MapWindowsSecurityContextOptionsKubeToAPI(v corev1.WindowsSecurityContextOptions) testkube.WindowsSecurityContextOptions {
	return testkube.WindowsSecurityContextOptions{
		GmsaCredentialSpecName: MapStringToBoxedString(v.GMSACredentialSpecName),
		GmsaCredentialSpec:     MapStringToBoxedString(v.GMSACredentialSpec),
		RunAsUserName:          MapStringToBoxedString(v.RunAsUserName),
		HostProcess:            MapBoolToBoxedBoolean(v.HostProcess),
	}
}

func MapSysctlKubeToAPI(v corev1.Sysctl) testkube.Sysctl {
	return testkube.Sysctl{
		Name:  v.Name,
		Value: v.Value,
	}
}

func MapSeccompProfileKubeToAPI(v corev1.SeccompProfile) testkube.SeccompProfile {
	return testkube.SeccompProfile{
		Type_:            string(v.Type),
		LocalhostProfile: MapStringToBoxedString(v.LocalhostProfile),
	}
}

func MapAppArmorProfileKubeToAPI(v corev1.AppArmorProfile) testkube.AppArmorProfile {
	return testkube.AppArmorProfile{
		Type_:            string(v.Type),
		LocalhostProfile: MapStringToBoxedString(v.LocalhostProfile),
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
		SidecarScraper:             v.SidecarScraper,
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
	return testkube.TestWorkflowTarballFilePattern{Static: v.Static}
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

func MapLabelSelectorRequirementToAPI(v metav1.LabelSelectorRequirement) testkube.LabelSelectorRequirement {
	return testkube.LabelSelectorRequirement{
		Key:      v.Key,
		Operator: string(v.Operator),
		Values:   v.Values,
	}
}

func MapSelectorToAPI(v metav1.LabelSelector) testkube.LabelSelector {
	return testkube.LabelSelector{
		MatchLabels:      v.MatchLabels,
		MatchExpressions: common.MapSlice(v.MatchExpressions, MapLabelSelectorRequirementToAPI),
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
		Selector:      common.MapPtr(v.Selector, MapSelectorToAPI),
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

func MapStepParallelFetchKubeToAPI(v testworkflowsv1.StepParallelFetch) testkube.TestWorkflowStepParallelFetch {
	return testkube.TestWorkflowStepParallelFetch{
		From:  v.From,
		To:    v.To,
		Files: common.MapPtr(v.Files, MapTarballFilePatternKubeToAPI),
	}
}

func MapStepParallelKubeToAPI(v testworkflowsv1.StepParallel) testkube.TestWorkflowStepParallel {
	return testkube.TestWorkflowStepParallel{
		Count:       MapIntOrStringToBoxedString(v.Count),
		MaxCount:    MapIntOrStringToBoxedString(v.MaxCount),
		Matrix:      MapDynamicListMapKubeToAPI(v.Matrix),
		Shards:      MapDynamicListMapKubeToAPI(v.Shards),
		Parallelism: v.Parallelism,
		Description: v.Description,
		Logs:        MapStringToBoxedString(v.Logs),
		Transfer:    common.MapSlice(v.Transfer, MapStepParallelTransferKubeToAPI),
		Fetch:       common.MapSlice(v.Fetch, MapStepParallelFetchKubeToAPI),
		Use:         common.MapSlice(v.Use, MapTemplateRefKubeToAPI),
		Config:      common.MapMap(v.Config, MapParameterSchemaKubeToAPI),
		System:      common.MapPtr(v.System, MapSystemKubeToAPI),
		Content:     common.MapPtr(v.Content, MapContentKubeToAPI),
		Container:   common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Job:         common.MapPtr(v.Job, MapJobConfigKubeToAPI),
		Pod:         common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
		Setup:       common.MapSlice(v.Setup, MapStepKubeToAPI),
		Steps:       common.MapSlice(v.Steps, MapStepKubeToAPI),
		After:       common.MapSlice(v.After, MapStepKubeToAPI),
		Paused:      v.Paused,
		Negative:    v.Negative,
		Optional:    v.Optional,
		Template:    common.MapPtr(v.Template, MapTemplateRefKubeToAPI),
		Retry:       common.MapPtr(v.Retry, MapRetryPolicyKubeToAPI),
		Timeout:     v.Timeout,
		Delay:       v.Delay,
		Shell:       v.Shell,
		Run:         common.MapPtr(v.Run, MapStepRunKubeToAPI),
		Execute:     common.MapPtr(v.Execute, MapStepExecuteKubeToAPI),
		Artifacts:   common.MapPtr(v.Artifacts, MapStepArtifactsKubeToAPI),
		Pvcs:        common.MapMap(v.Pvcs, MapPvcConfigKubeToAPI),
	}
}

func MapIndependentStepParallelKubeToAPI(v testworkflowsv1.IndependentStepParallel) testkube.TestWorkflowIndependentStepParallel {
	return testkube.TestWorkflowIndependentStepParallel{
		Count:       MapIntOrStringToBoxedString(v.Count),
		MaxCount:    MapIntOrStringToBoxedString(v.MaxCount),
		Matrix:      MapDynamicListMapKubeToAPI(v.Matrix),
		Shards:      MapDynamicListMapKubeToAPI(v.Shards),
		Parallelism: v.Parallelism,
		Description: v.Description,
		Logs:        MapStringToBoxedString(v.Logs),
		Transfer:    common.MapSlice(v.Transfer, MapStepParallelTransferKubeToAPI),
		Fetch:       common.MapSlice(v.Fetch, MapStepParallelFetchKubeToAPI),
		Config:      common.MapMap(v.Config, MapParameterSchemaKubeToAPI),
		System:      common.MapPtr(v.System, MapSystemKubeToAPI),
		Content:     common.MapPtr(v.Content, MapContentKubeToAPI),
		Container:   common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Job:         common.MapPtr(v.Job, MapJobConfigKubeToAPI),
		Pod:         common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
		Setup:       common.MapSlice(v.Setup, MapIndependentStepKubeToAPI),
		Steps:       common.MapSlice(v.Steps, MapIndependentStepKubeToAPI),
		After:       common.MapSlice(v.After, MapIndependentStepKubeToAPI),
		Paused:      v.Paused,
		Negative:    v.Negative,
		Optional:    v.Optional,
		Retry:       common.MapPtr(v.Retry, MapRetryPolicyKubeToAPI),
		Timeout:     v.Timeout,
		Delay:       v.Delay,
		Shell:       v.Shell,
		Run:         common.MapPtr(v.Run, MapStepRunKubeToAPI),
		Execute:     common.MapPtr(v.Execute, MapStepExecuteKubeToAPI),
		Artifacts:   common.MapPtr(v.Artifacts, MapStepArtifactsKubeToAPI),
		Pvcs:        common.MapMap(v.Pvcs, MapPvcConfigKubeToAPI),
	}
}

func MapExecActionKubeToAPI(v corev1.ExecAction) testkube.ExecAction {
	return testkube.ExecAction{
		Command: v.Command,
	}
}

func MapHTTPHeaderKubeToAPI(v corev1.HTTPHeader) testkube.HttpHeader {
	return testkube.HttpHeader{
		Name:  v.Name,
		Value: v.Value,
	}
}

func MapHTTPGetActionKubeToAPI(v corev1.HTTPGetAction) testkube.HttpGetAction {
	return testkube.HttpGetAction{
		Path:        v.Path,
		Port:        MapIntOrStringToString(v.Port),
		Host:        v.Host,
		Scheme:      string(v.Scheme),
		HttpHeaders: common.MapSlice(v.HTTPHeaders, MapHTTPHeaderKubeToAPI),
	}
}

func MapTCPSocketActionKubeToAPI(v corev1.TCPSocketAction) testkube.TcpSocketAction {
	return testkube.TcpSocketAction{
		Port: MapIntOrStringToString(v.Port),
		Host: v.Host,
	}
}

func MapGRPCActionKubeToAPI(v corev1.GRPCAction) testkube.GrpcAction {
	return testkube.GrpcAction{
		Port:    v.Port,
		Service: MapStringToBoxedString(v.Service),
	}
}

func MapProbeKubeToAPI(v corev1.Probe) testkube.Probe {
	return testkube.Probe{
		InitialDelaySeconds:           v.InitialDelaySeconds,
		TimeoutSeconds:                v.TimeoutSeconds,
		PeriodSeconds:                 v.PeriodSeconds,
		SuccessThreshold:              v.SuccessThreshold,
		FailureThreshold:              v.FailureThreshold,
		TerminationGracePeriodSeconds: MapInt64ToBoxedInteger(v.TerminationGracePeriodSeconds),
		Exec:                          common.MapPtr(v.Exec, MapExecActionKubeToAPI),
		HttpGet:                       common.MapPtr(v.HTTPGet, MapHTTPGetActionKubeToAPI),
		TcpSocket:                     common.MapPtr(v.TCPSocket, MapTCPSocketActionKubeToAPI),
		Grpc:                          common.MapPtr(v.GRPC, MapGRPCActionKubeToAPI),
	}
}

func MapIndependentServiceSpecKubeToAPI(v testworkflowsv1.IndependentServiceSpec) testkube.TestWorkflowIndependentServiceSpec {
	return testkube.TestWorkflowIndependentServiceSpec{
		Count:           MapIntOrStringToBoxedString(v.Count),
		MaxCount:        MapIntOrStringToBoxedString(v.MaxCount),
		Matrix:          MapDynamicListMapKubeToAPI(v.Matrix),
		Shards:          MapDynamicListMapKubeToAPI(v.Shards),
		Description:     v.Description,
		Pod:             common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
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
		Timeout:         v.Timeout,
		Transfer:        common.MapSlice(v.Transfer, MapStepParallelTransferKubeToAPI),
		Content:         common.MapPtr(v.Content, MapContentKubeToAPI),
		Logs:            MapStringToBoxedString(v.Logs),
		RestartPolicy:   string(v.RestartPolicy),
		ReadinessProbe:  common.MapPtr(v.ReadinessProbe, MapProbeKubeToAPI),
		Pvcs:            common.MapMap(v.Pvcs, MapPvcConfigKubeToAPI),
	}
}

func MapServiceSpecKubeToAPI(v testworkflowsv1.ServiceSpec) testkube.TestWorkflowServiceSpec {
	return testkube.TestWorkflowServiceSpec{
		Count:           MapIntOrStringToBoxedString(v.Count),
		MaxCount:        MapIntOrStringToBoxedString(v.MaxCount),
		Matrix:          MapDynamicListMapKubeToAPI(v.Matrix),
		Shards:          MapDynamicListMapKubeToAPI(v.Shards),
		Use:             common.MapSlice(v.Use, MapTemplateRefKubeToAPI),
		Description:     v.Description,
		Pod:             common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
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
		Timeout:         v.Timeout,
		Transfer:        common.MapSlice(v.Transfer, MapStepParallelTransferKubeToAPI),
		Content:         common.MapPtr(v.Content, MapContentKubeToAPI),
		Logs:            MapStringToBoxedString(v.Logs),
		RestartPolicy:   string(v.RestartPolicy),
		ReadinessProbe:  common.MapPtr(v.ReadinessProbe, MapProbeKubeToAPI),
		Pvcs:            common.MapMap(v.Pvcs, MapPvcConfigKubeToAPI),
	}
}

func MapStepKubeToAPI(v testworkflowsv1.Step) testkube.TestWorkflowStep {
	return testkube.TestWorkflowStep{
		Name:       v.Name,
		Condition:  v.Condition,
		Paused:     v.Paused,
		Negative:   v.Negative,
		Optional:   v.Optional,
		Pure:       MapBoolToBoxedBoolean(v.Pure),
		Use:        common.MapSlice(v.Use, MapTemplateRefKubeToAPI),
		Template:   common.MapPtr(v.Template, MapTemplateRefKubeToAPI),
		Retry:      common.MapPtr(v.Retry, MapRetryPolicyKubeToAPI),
		Timeout:    v.Timeout,
		Delay:      v.Delay,
		Content:    common.MapPtr(v.Content, MapContentKubeToAPI),
		Services:   common.MapMap(v.Services, MapServiceSpecKubeToAPI),
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
		Pure:       MapBoolToBoxedBoolean(v.Pure),
		Retry:      common.MapPtr(v.Retry, MapRetryPolicyKubeToAPI),
		Timeout:    v.Timeout,
		Delay:      v.Delay,
		Content:    common.MapPtr(v.Content, MapContentKubeToAPI),
		Services:   common.MapMap(v.Services, MapIndependentServiceSpecKubeToAPI),
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

func MapSystemKubeToAPI(v testworkflowsv1.TestWorkflowSystem) testkube.TestWorkflowSystem {
	return testkube.TestWorkflowSystem{
		PureByDefault:      MapBoolToBoxedBoolean(v.PureByDefault),
		IsolatedContainers: MapBoolToBoxedBoolean(v.IsolatedContainers),
	}
}

func MapSpecKubeToAPI(v testworkflowsv1.TestWorkflowSpec) testkube.TestWorkflowSpec {
	return testkube.TestWorkflowSpec{
		Use:       common.MapSlice(v.Use, MapTemplateRefKubeToAPI),
		Config:    common.MapMap(v.Config, MapParameterSchemaKubeToAPI),
		System:    common.MapPtr(v.System, MapSystemKubeToAPI),
		Content:   common.MapPtr(v.Content, MapContentKubeToAPI),
		Services:  common.MapMap(v.Services, MapServiceSpecKubeToAPI),
		Container: common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Job:       common.MapPtr(v.Job, MapJobConfigKubeToAPI),
		Pod:       common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
		Setup:     common.MapSlice(v.Setup, MapStepKubeToAPI),
		Steps:     common.MapSlice(v.Steps, MapStepKubeToAPI),
		After:     common.MapSlice(v.After, MapStepKubeToAPI),
		Events:    common.MapSlice(v.Events, MapEventKubeToAPI),
		Execution: common.MapPtr(v.Execution, MapTestWorkflowTagSchemaKubeToAPI),
		Pvcs:      common.MapMap(v.Pvcs, MapPvcConfigKubeToAPI),
	}
}

func MapTemplateSpecKubeToAPI(v testworkflowsv1.TestWorkflowTemplateSpec) testkube.TestWorkflowTemplateSpec {
	return testkube.TestWorkflowTemplateSpec{
		Config:    common.MapMap(v.Config, MapParameterSchemaKubeToAPI),
		System:    common.MapPtr(v.System, MapSystemKubeToAPI),
		Content:   common.MapPtr(v.Content, MapContentKubeToAPI),
		Services:  common.MapMap(v.Services, MapIndependentServiceSpecKubeToAPI),
		Container: common.MapPtr(v.Container, MapContainerConfigKubeToAPI),
		Job:       common.MapPtr(v.Job, MapJobConfigKubeToAPI),
		Pod:       common.MapPtr(v.Pod, MapPodConfigKubeToAPI),
		Setup:     common.MapSlice(v.Setup, MapIndependentStepKubeToAPI),
		Steps:     common.MapSlice(v.Steps, MapIndependentStepKubeToAPI),
		After:     common.MapSlice(v.After, MapIndependentStepKubeToAPI),
		Events:    common.MapSlice(v.Events, MapEventKubeToAPI),
		Execution: common.MapPtr(v.Execution, MapTestWorkflowTagSchemaKubeToAPI),
		Pvcs:      common.MapMap(v.Pvcs, MapPvcConfigKubeToAPI),
	}
}

func MapTestWorkflowKubeToAPI(w testworkflowsv1.TestWorkflow) testkube.TestWorkflow {
	updateTime := w.CreationTimestamp.Time
	if w.DeletionTimestamp != nil {
		updateTime = w.DeletionTimestamp.Time
	} else {
		for _, field := range w.ManagedFields {
			if field.Time != nil && field.Time.After(updateTime) {
				updateTime = field.Time.Time
			}
		}
	}

	return testkube.TestWorkflow{
		Name:        w.Name,
		Namespace:   w.Namespace,
		Labels:      w.Labels,
		Annotations: w.Annotations,
		Created:     w.CreationTimestamp.Time,
		Updated:     updateTime,
		Description: w.Description,
		Spec:        common.Ptr(MapSpecKubeToAPI(w.Spec)),
	}
}

func MapTestWorkflowTemplateKubeToAPI(w testworkflowsv1.TestWorkflowTemplate) testkube.TestWorkflowTemplate {
	updateTime := w.CreationTimestamp.Time
	if w.DeletionTimestamp != nil {
		updateTime = w.DeletionTimestamp.Time
	} else {
		for _, field := range w.ManagedFields {
			if field.Time != nil && field.Time.After(updateTime) {
				updateTime = field.Time.Time
			}
		}
	}

	return testkube.TestWorkflowTemplate{
		Name:        w.Name,
		Namespace:   w.Namespace,
		Labels:      w.Labels,
		Annotations: w.Annotations,
		Created:     w.CreationTimestamp.Time,
		Updated:     updateTime,
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

func MapTestWorkflowTagSchemaKubeToAPI(v testworkflowsv1.TestWorkflowTagSchema) testkube.TestWorkflowTagSchema {
	return testkube.TestWorkflowTagSchema{
		Tags:   v.Tags,
		Target: common.MapPtr(v.Target, MapTargetKubeToAPI),
	}
}

func MapTypeLocalObjectReferenceKubeToAPI(v corev1.TypedLocalObjectReference) testkube.TypedLocalObjectReference {
	return testkube.TypedLocalObjectReference{
		ApiGroup: MapStringToBoxedString(v.APIGroup),
		Kind:     v.Kind,
		Name:     v.Name,
	}
}

func MapTypeObjectReferenceKubeToAPI(v corev1.TypedObjectReference) testkube.TypedObjectReference {
	return testkube.TypedObjectReference{
		ApiGroup:  MapStringToBoxedString(v.APIGroup),
		Kind:      v.Kind,
		Name:      v.Name,
		Namespace: MapStringToBoxedString(v.Namespace),
	}
}

func MapVolumeResourceRequirementsKubeToAPI(v corev1.VolumeResourceRequirements) *testkube.TestWorkflowResources {
	return &testkube.TestWorkflowResources{
		Limits:   MapResourcesListKubeCoreToAPI(v.Limits),
		Requests: MapResourcesListKubeCoreToAPI(v.Requests),
	}
}

func MapResourcesListKubeCoreToAPI(v corev1.ResourceList) *testkube.TestWorkflowResourcesList {
	if len(v) == 0 {
		return nil
	}

	res := &testkube.TestWorkflowResourcesList{}
	if q, ok := v[corev1.ResourceCPU]; ok {
		res.Cpu = q.String()
	}

	if q, ok := v[corev1.ResourceMemory]; ok {
		res.Memory = q.String()
	}

	if q, ok := v[corev1.ResourceStorage]; ok {
		res.Storage = q.String()
	}

	if q, ok := v[corev1.ResourceEphemeralStorage]; ok {
		res.EphemeralStorage = q.String()
	}

	return res
}

func MapPvcConfigKubeToAPI(v corev1.PersistentVolumeClaimSpec) testkube.TestWorkflowPvcConfig {
	return testkube.TestWorkflowPvcConfig{
		AccessModes: common.MapSlice(v.AccessModes,
			func(v corev1.PersistentVolumeAccessMode) string { return (string)(v) }),
		VolumeMode:                MapStringToBoxedString((*string)(v.VolumeMode)),
		Resources:                 MapVolumeResourceRequirementsKubeToAPI(v.Resources),
		StorageClassName:          MapStringToBoxedString(v.StorageClassName),
		VolumeName:                v.VolumeName,
		Selector:                  common.MapPtr(v.Selector, MapSelectorToAPI),
		DataSource:                common.MapPtr(v.DataSource, MapTypeLocalObjectReferenceKubeToAPI),
		DataSourceRef:             common.MapPtr(v.DataSourceRef, MapTypeObjectReferenceKubeToAPI),
		VolumeAttributesClassName: MapStringToBoxedString(v.VolumeAttributesClassName),
	}
}

func MapTestWorkflowExecutionResourceAggregationsReportKubeToAPI(
	v *testworkflowsv1.TestWorkflowExecutionResourceAggregationsReport,
) *testkube.TestWorkflowExecutionResourceAggregationsReport {
	if v == nil {
		return nil
	}
	return &testkube.TestWorkflowExecutionResourceAggregationsReport{
		Global: MapTestWorkflowExecutionResourceAggregationsByMeasurementKubeToAPI(v.Global),
		Step:   MapTestWorkflowExecutionStepResourceAggregationsKubeToAPI(v.Step),
	}
}

func MapTestWorkflowExecutionStepResourceAggregationsKubeToAPI(
	vs []*testworkflowsv1.TestWorkflowExecutionStepResourceAggregations,
) []testkube.TestWorkflowExecutionStepResourceAggregations {
	r := make([]testkube.TestWorkflowExecutionStepResourceAggregations, 0, len(vs))

	for _, v := range vs {
		r = append(r, testkube.TestWorkflowExecutionStepResourceAggregations{
			Ref:          v.Ref,
			Aggregations: MapTestWorkflowExecutionResourceAggregationsByMeasurementKubeToAPI(v.Aggregations),
		})
	}

	return r
}

func MapTestWorkflowExecutionResourceAggregationsByMeasurementKubeToAPI(
	v testworkflowsv1.TestWorkflowExecutionResourceAggregationsByMeasurement,
) map[string]map[string]testkube.TestWorkflowExecutionResourceAggregations {
	result := make(map[string]map[string]testkube.TestWorkflowExecutionResourceAggregations)

	for measurement, byField := range v {
		if _, ok := result[measurement]; !ok {
			result[measurement] = make(map[string]testkube.TestWorkflowExecutionResourceAggregations)
		}
		for field, wrapper := range byField {
			apiWrapper := MapTestWorkflowExecutionResourceAggregationsKubeToAPI(wrapper)
			if apiWrapper != nil {
				result[measurement][field] = *apiWrapper
			} else {
				// If needed, handle the case where wrapper is nil or conversion is nil
				result[measurement][field] = testkube.TestWorkflowExecutionResourceAggregations{}
			}
		}
	}

	return result
}

func MapTestWorkflowExecutionResourceAggregationsKubeToAPI(
	v *testworkflowsv1.TestWorkflowExecutionResourceAggregations,
) *testkube.TestWorkflowExecutionResourceAggregations {
	if v == nil {
		return nil
	}
	return &testkube.TestWorkflowExecutionResourceAggregations{
		Total:  v.Total,
		Min:    v.Min,
		Max:    v.Max,
		Avg:    v.Avg,
		StdDev: v.StdDev,
	}
}
